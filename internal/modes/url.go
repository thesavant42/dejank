package modes

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesavant42/dejank/internal/assets"
	"github.com/thesavant42/dejank/internal/sourcemap"
	"github.com/thesavant42/dejank/internal/ui"

	"golang.org/x/net/html"
)

// URLResult contains the results of processing a URL.
type URLResult struct {
	URL             string
	ScriptsFound    int
	ScriptsProcessed int
	SourcesRestored int
	AssetsExtracted int
	Errors          []error
}

// RunURL crawls a webpage, downloads scripts and sourcemaps, and restores sources.
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

	// Fetch the HTML page
	htmlContent, err := cfg.Client.Get(targetURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}

	// Extract script URLs
	scriptURLs := extractScriptURLs(htmlContent, targetURL)
	result.ScriptsFound = len(scriptURLs)

	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Found %d script(s) on %s", len(scriptURLs), targetURL)))
	}
	cfg.emit("scripts_found", result.ScriptsFound)

	// Process each script
	for i, scriptURL := range scriptURLs {
		cfg.emit("processing_script", map[string]interface{}{
			"index": i,
			"total": len(scriptURLs),
			"url":   scriptURL,
		})
		if err := processScript(cfg, scriptURL, paths, result); err != nil {
			result.Errors = append(result.Errors, err)
		} else {
			result.ScriptsProcessed++
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

// extractScriptURLs parses HTML and returns all script src URLs.
func extractScriptURLs(htmlContent, baseURL string) []string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return nil
	}

	var urls []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "src" && attr.Val != "" {
					resolved, err := resolveURL(baseURL, attr.Val)
					if err == nil {
						urls = append(urls, resolved)
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)

	return urls
}

// processScript downloads a script, finds its sourcemap, and restores sources.
func processScript(cfg *Config, scriptURL string, paths DomainPaths, result *URLResult) error {
	filename := filenameFromURL(scriptURL)
	scriptPath := filepath.Join(paths.DownloadedSite, filename)

	// Download the script
	if err := cfg.Client.Download(scriptURL, scriptPath); err != nil {
		return fmt.Errorf("failed to download %s: %w", scriptURL, err)
	}

	if cfg.Verbose {
		fmt.Println(ui.Success(fmt.Sprintf("Downloaded: %s", filename)))
	}

	// Read script content
	content, err := os.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read downloaded script: %w", err)
	}

	jsContent := string(content)

	// Check for inline sourcemap first
	if sourcemap.HasInlineSourceMap(jsContent) {
		sm, err := sourcemap.ExtractInlineSourceMap(jsContent)
		if err != nil {
			return fmt.Errorf("failed to extract inline sourcemap: %w", err)
		}
		if sm != nil {
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

	// Look for external sourcemap URL
	mapURL := sourcemap.ExtractSourceMappingURL(jsContent)
	if mapURL == "" {
		if cfg.Verbose {
			fmt.Println(ui.Warning(fmt.Sprintf("No sourcemap found for: %s", filename)))
		}
		return nil
	}

	// Resolve relative map URL
	resolvedMapURL, err := resolveURL(scriptURL, mapURL)
	if err != nil {
		return fmt.Errorf("failed to resolve map URL: %w", err)
	}

	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Found sourcemap: %s", resolvedMapURL)))
	}

	// Download the sourcemap
	mapFilename := filenameFromURL(resolvedMapURL)
	mapPath := filepath.Join(paths.DownloadedSite, mapFilename)

	if err := cfg.Client.Download(resolvedMapURL, mapPath); err != nil {
		return fmt.Errorf("failed to download sourcemap: %w", err)
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

