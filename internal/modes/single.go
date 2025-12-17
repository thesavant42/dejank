package modes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesavant42/dejank/internal/sourcemap"
	"github.com/thesavant42/dejank/internal/ui"
)

// SingleResult contains the results of processing a single script URL.
type SingleResult struct {
	URL             string
	SourcesRestored int
	MapFound        bool
	Errors          []error
}

// RunSingle downloads a single script URL, finds its sourcemap, and restores sources.
func RunSingle(cfg *Config, scriptURL string) (*SingleResult, error) {
	// Require scheme
	if !strings.HasPrefix(scriptURL, "http://") && !strings.HasPrefix(scriptURL, "https://") {
		return nil, fmt.Errorf("invalid URL: must include http:// or https:// scheme")
	}

	result := &SingleResult{URL: scriptURL}

	// Parse URL to get hostname
	parsed, err := url.Parse(scriptURL)
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

	// Download the script
	filename := filenameFromURL(scriptURL)
	scriptPath := filepath.Join(paths.DownloadedSite, filename)

	if err := cfg.Client.Download(scriptURL, scriptPath); err != nil {
		return nil, fmt.Errorf("failed to download script: %w", err)
	}

	if cfg.Verbose {
		fmt.Println(ui.Success(fmt.Sprintf("Downloaded: %s", filename)))
	}

	// Read script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read downloaded script: %w", err)
	}

	jsContent := string(content)

	// Check for inline sourcemap first
	if sourcemap.HasInlineSourceMap(jsContent) {
		sm, err := sourcemap.ExtractInlineSourceMap(jsContent)
		if err != nil {
			return nil, fmt.Errorf("failed to extract inline sourcemap: %w", err)
		}
		if sm != nil {
			result.MapFound = true

			// Save the inline map for reference
			mapPath := scriptPath + ".inline.map"
			mapJSON, _ := json.MarshalIndent(sm, "", "  ")
			os.WriteFile(mapPath, mapJSON, 0644)

			if cfg.Verbose {
				fmt.Println(ui.Success(fmt.Sprintf("Extracted inline sourcemap: %s", filepath.Base(mapPath))))
			}

			restoreResult := sourcemap.RestoreSources(sm, paths.RestoredSources)
			result.SourcesRestored = restoreResult.RestoredCount
			result.Errors = restoreResult.Errors
			return result, nil
		}
	}

	// Look for external sourcemap URL
	mapURL := sourcemap.ExtractSourceMappingURL(jsContent)
	if mapURL == "" {
		if cfg.Verbose {
			fmt.Println(ui.Warning(fmt.Sprintf("No sourcemap found in: %s", filename)))
		}
		return result, nil
	}

	result.MapFound = true

	// Resolve relative map URL
	resolvedMapURL, err := resolveURL(scriptURL, mapURL)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve map URL: %w", err)
	}

	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Found sourcemap: %s", resolvedMapURL)))
	}

	// Download the sourcemap
	mapFilename := filenameFromURL(resolvedMapURL)
	mapPath := filepath.Join(paths.DownloadedSite, mapFilename)

	if err := cfg.Client.Download(resolvedMapURL, mapPath); err != nil {
		return nil, fmt.Errorf("failed to download sourcemap: %w", err)
	}

	if cfg.Verbose {
		fmt.Println(ui.Success(fmt.Sprintf("Downloaded: %s", mapFilename)))
	}

	// Parse and restore
	sm, err := sourcemap.ParseFile(mapPath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse sourcemap: %w", err)
	}

	restoreResult := sourcemap.RestoreSources(sm, paths.RestoredSources)
	result.SourcesRestored = restoreResult.RestoredCount
	result.Errors = restoreResult.Errors

	return result, nil
}

