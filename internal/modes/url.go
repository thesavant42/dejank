package modes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesavant42/dejank/internal/assets"
	"github.com/thesavant42/dejank/internal/envars"
	"github.com/thesavant42/dejank/internal/fetch"
	"github.com/thesavant42/dejank/internal/sourcemap"
	"github.com/thesavant42/dejank/internal/ui"
)

// URLResult contains the results of processing a URL.
type URLResult struct {
	URL              string
	ScriptsFound     int
	MapsDiscovered   int
	SourcesRestored  int
	AssetsExtracted  int
	EnvVarsExtracted int
	Errors           []error
}

// RunURL crawls a webpage using headless Chrome, discovers all scripts and sourcemaps,
// and restores sources.
func RunURL(cfg *Config, targetURL string) (*URLResult, error) {
	// Require scheme
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		return nil, fmt.Errorf("invalid URL: must include http:// or https:// scheme")
	}

	result := &URLResult{URL: targetURL}

	// Parse URL to get hostname
	parsed, err := url.Parse(targetURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	paths := GetDomainPaths(cfg.OutputRoot, parsed.Host)

	// Check for existing directory
	if paths.Exists() && !cfg.Force {
		return nil, fmt.Errorf("output directory already exists: %s (use -f to overwrite)", paths.Base)
	}

	if err := paths.EnsureDirs(); err != nil {
		return nil, err
	}

	// Use browser client to discover resources via JS execution
	if cfg.Verbose {
		fmt.Println(ui.Info("Launching headless browser..."))
	}

	browser := fetch.NewBrowserClient()
	discovered, err := browser.DiscoverResources(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to discover resources: %w", err)
	}

	result.ScriptsFound = len(discovered.Scripts)

	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Discovered %d scripts via browser", result.ScriptsFound)))
	}

	cfg.emit("discovery_complete", map[string]int{
		"scripts": result.ScriptsFound,
	})

	// Track discovered maps to avoid duplicates
	processedMaps := make(map[string]bool)

	// Process sourcemaps discovered via network interception and response headers
	for _, mapURL := range discovered.SourceMaps {
		if processedMaps[mapURL] {
			continue
		}
		processedMaps[mapURL] = true

		if cfg.Verbose {
			fmt.Println(ui.Info(fmt.Sprintf("Processing discovered sourcemap: %s", mapURL)))
		}

		if err := processSourceMap(cfg, mapURL, paths, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// Process scripts to find additional sourcemaps via inline/header references
	for i, scriptURL := range discovered.Scripts {
		cfg.emit("processing_script", map[string]interface{}{
			"index": i,
			"total": len(discovered.Scripts),
			"url":   scriptURL,
		})

		if err := processScriptForMaps(cfg, scriptURL, paths, result, processedMaps); err != nil {
			result.Errors = append(result.Errors, err)
		}
	}

	// MapsDiscovered is the count of unique maps we found and processed
	result.MapsDiscovered = len(processedMaps)

	// Extract environment variables from all downloaded JS files
	if cfg.Verbose {
		fmt.Println(ui.Info("Extracting environment variables from bundled JS..."))
	}
	allEnvVars := make(map[string]string)
	entries, err := os.ReadDir(paths.DownloadedSite)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".js") {
				continue
			}
			jsPath := filepath.Join(paths.DownloadedSite, entry.Name())
			content, err := os.ReadFile(jsPath)
			if err != nil {
				continue
			}
			extractedVars := envars.ExtractEnvVars(string(content))
			allEnvVars = envars.MergeEnvVars(allEnvVars, extractedVars)
		}
	}

	// Write .env file if we found any environment variables
	if len(allEnvVars) > 0 {
		envPath := filepath.Join(paths.RestoredSources, ".env")
		if err := envars.WriteEnvFile(allEnvVars, envPath); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to write .env file: %w", err))
		} else {
			result.EnvVarsExtracted = len(allEnvVars)
			if cfg.Verbose {
				fmt.Println(ui.Success(fmt.Sprintf("Extracted %d environment variable(s) to .env", len(allEnvVars))))
			}
		}
	}

	// Extract embedded assets from restored sources
	if cfg.Verbose {
		fmt.Println(ui.Info("Scanning for embedded base64 assets..."))
	}
	assetResult := assets.ExtractFromDirectory(paths.RestoredSources, paths.ExtractedAssets)
	result.AssetsExtracted = assetResult.ExtractedCount
	result.Errors = append(result.Errors, assetResult.Errors...)

	return result, nil
}

// processSourceMap downloads and processes a sourcemap URL.
func processSourceMap(cfg *Config, mapURL string, paths DomainPaths, result *URLResult) error {
	mapFilename := filenameFromURL(mapURL)
	mapPath := filepath.Join(paths.DownloadedSite, mapFilename)

	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Downloading sourcemap: %s", mapFilename)))
	}

	if err := cfg.Client.Download(mapURL, mapPath); err != nil {
		return fmt.Errorf("failed to download sourcemap %s: %w", mapURL, err)
	}

	if cfg.Verbose {
		fmt.Println(ui.Success(fmt.Sprintf("Downloaded: %s", mapFilename)))
	}

	// Parse and restore
	sm, err := sourcemap.ParseFile(mapPath)
	if err != nil {
		return fmt.Errorf("failed to parse sourcemap: %w", err)
	}

	restoreResult := sourcemap.RestoreSources(sm, paths.RestoredSources)
	result.SourcesRestored += restoreResult.RestoredCount
	result.Errors = append(result.Errors, restoreResult.Errors...)

	return nil
}

// processScriptForMaps downloads a script and checks for inline/external sourcemaps
// that weren't caught by network interception.
func processScriptForMaps(cfg *Config, scriptURL string, paths DomainPaths, result *URLResult, processedMaps map[string]bool) error {
	filename := filenameFromURL(scriptURL)
	scriptPath := filepath.Join(paths.DownloadedSite, filename)

	// Download the script
	if err := cfg.Client.Download(scriptURL, scriptPath); err != nil {
		return fmt.Errorf("failed to download %s: %w", scriptURL, err)
	}

	// Read script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read downloaded script: %w", err)
	}

	jsContent := string(content)

	// Check for inline sourcemap first
	if sourcemap.HasInlineSourceMap(jsContent) {
		// Use script URL as unique key for inline maps
		inlineKey := scriptURL + ":inline"
		if processedMaps[inlineKey] {
			return nil
		}

		sm, err := sourcemap.ExtractInlineSourceMap(jsContent)
		if err != nil {
			return fmt.Errorf("failed to extract inline sourcemap: %w", err)
		}
		if sm != nil {
			processedMaps[inlineKey] = true

			// Save the inline map for reference
			mapPath := scriptPath + ".inline.map"
			mapJSON, _ := json.MarshalIndent(sm, "", "  ")
			os.WriteFile(mapPath, mapJSON, 0644)

			if cfg.Verbose {
				fmt.Println(ui.Success(fmt.Sprintf("Extracted inline sourcemap: %s", filepath.Base(mapPath))))
			}

			restoreResult := sourcemap.RestoreSources(sm, paths.RestoredSources)
			result.SourcesRestored += restoreResult.RestoredCount
			result.Errors = append(result.Errors, restoreResult.Errors...)
			return nil
		}
	}

	// Look for external sourcemap URL that wasn't caught by network interception
	mapURL := sourcemap.ExtractSourceMappingURL(jsContent)
	if mapURL == "" {
		return nil
	}

	// Resolve relative map URL
	resolvedMapURL, err := resolveURL(scriptURL, mapURL)
	if err != nil {
		return fmt.Errorf("failed to resolve map URL: %w", err)
	}

	// Skip if already processed
	if processedMaps[resolvedMapURL] {
		return nil
	}
	processedMaps[resolvedMapURL] = true

	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Found additional sourcemap: %s", resolvedMapURL)))
	}

	// Process this map
	if err := processSourceMap(cfg, resolvedMapURL, paths, result); err != nil {
		return err
	}

	return nil
}
