package modes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesavant42/dejank/internal/assets"
	"github.com/thesavant42/dejank/internal/sourcemap"
	"github.com/thesavant42/dejank/internal/ui"
)

// LocalResult contains the results of processing local files.
type LocalResult struct {
	TargetsProcessed int
	MapsProcessed    int
	SourcesRestored  int
	AssetsExtracted  int
	Errors           []error
}

// RunLocal processes local .js and .map files in the output directory.
// If target is empty, processes all domain directories under outputRoot.
// If target is specified, processes only that directory.
func RunLocal(cfg *Config, target string) (*LocalResult, error) {
	result := &LocalResult{}

	var targets []string

	if target != "" {
		// Single target specified
		absTarget, err := filepath.Abs(target)
		if err != nil {
			return nil, fmt.Errorf("invalid target path: %w", err)
		}
		targets = append(targets, absTarget)
	} else {
		// Find all domain directories in output root
		entries, err := os.ReadDir(cfg.OutputRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to read output directory: %w", err)
		}

		for _, entry := range entries {
			if entry.IsDir() {
				targets = append(targets, filepath.Join(cfg.OutputRoot, entry.Name()))
			}
		}
	}

	if len(targets) == 0 {
		return result, nil
	}

	for _, domainPath := range targets {
		if err := processLocalDomain(cfg, domainPath, result); err != nil {
			result.Errors = append(result.Errors, err)
		}
		result.TargetsProcessed++
	}

	return result, nil
}

// processLocalDomain processes a single domain directory.
func processLocalDomain(cfg *Config, domainPath string, result *LocalResult) error {
	domain := filepath.Base(domainPath)
	downloadDir := filepath.Join(domainPath, "downloaded_site")
	restoreDir := filepath.Join(domainPath, "restored_sources")
	assetsDir := filepath.Join(domainPath, "extracted_assets")

	// Check if downloaded_site exists
	if _, err := os.Stat(downloadDir); os.IsNotExist(err) {
		if cfg.Verbose {
			fmt.Println(ui.Warning(fmt.Sprintf("Skipping %s: no downloaded_site folder", domain)))
		}
		return nil
	}

	// Ensure output directories exist
	os.MkdirAll(restoreDir, 0755)
	os.MkdirAll(assetsDir, 0755)

	// Read files in downloaded_site
	entries, err := os.ReadDir(downloadDir)
	if err != nil {
		return fmt.Errorf("failed to read download directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		fullPath := filepath.Join(downloadDir, filename)

		// Process .map files
		if strings.HasSuffix(filename, ".map") {
			if err := processMapFile(cfg, fullPath, restoreDir, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}

		// Process .js files (check for inline sourcemaps)
		if strings.HasSuffix(filename, ".js") {
			if err := processJSFile(cfg, fullPath, downloadDir, restoreDir, result); err != nil {
				result.Errors = append(result.Errors, err)
			}
		}
	}

	// Extract embedded assets
	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Scanning for embedded assets in: %s", restoreDir)))
	}
	assetResult := assets.ExtractFromDirectory(restoreDir, assetsDir)
	result.AssetsExtracted += assetResult.ExtractedCount
	result.Errors = append(result.Errors, assetResult.Errors...)

	if cfg.Verbose && assetResult.ExtractedCount > 0 {
		fmt.Println(ui.Success(fmt.Sprintf("Extracted %d asset(s)", assetResult.ExtractedCount)))
	}

	return nil
}

// processMapFile parses a .map file and restores sources.
func processMapFile(cfg *Config, mapPath, restoreDir string, result *LocalResult) error {
	if cfg.Verbose {
		fmt.Println(ui.Info(fmt.Sprintf("Processing: %s", filepath.Base(mapPath))))
	}

	sm, err := sourcemap.ParseFile(mapPath)
	if err != nil {
		return fmt.Errorf("failed to parse %s: %w", filepath.Base(mapPath), err)
	}

	restoreResult := sourcemap.RestoreSources(sm, restoreDir)
	result.MapsProcessed++
	result.SourcesRestored += restoreResult.RestoredCount
	result.Errors = append(result.Errors, restoreResult.Errors...)

	if cfg.Verbose {
		fmt.Println(ui.Success(fmt.Sprintf("Restored %d source(s) from %s", restoreResult.RestoredCount, filepath.Base(mapPath))))
	}

	return nil
}

// processJSFile checks for inline sourcemaps and extracts them.
func processJSFile(cfg *Config, jsPath, downloadDir, restoreDir string, result *LocalResult) error {
	content, err := os.ReadFile(jsPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", filepath.Base(jsPath), err)
	}

	jsContent := string(content)

	if !sourcemap.HasInlineSourceMap(jsContent) {
		return nil
	}

	sm, err := sourcemap.ExtractInlineSourceMap(jsContent)
	if err != nil {
		return fmt.Errorf("failed to extract inline sourcemap from %s: %w", filepath.Base(jsPath), err)
	}

	if sm == nil {
		return nil
	}

	// Save the extracted sourcemap
	mapPath := jsPath + ".inline.map"
	mapJSON, _ := json.MarshalIndent(sm, "", "  ")
	if err := os.WriteFile(mapPath, mapJSON, 0644); err != nil {
		return fmt.Errorf("failed to save inline map: %w", err)
	}

	if cfg.Verbose {
		fmt.Println(ui.Success(fmt.Sprintf("Extracted inline sourcemap: %s", filepath.Base(mapPath))))
	}

	// Restore sources
	restoreResult := sourcemap.RestoreSources(sm, restoreDir)
	result.MapsProcessed++
	result.SourcesRestored += restoreResult.RestoredCount
	result.Errors = append(result.Errors, restoreResult.Errors...)

	return nil
}

