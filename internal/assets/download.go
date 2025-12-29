// Package assets handles extraction and downloading of assets from restored source files.
package assets

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/thesavant42/dejank/internal/fetch"
)

var (
	// Matches various webpack asset export patterns:
	// export default __webpack_public_path__ + "static/media/filename.hash.ext";
	// export default __webpack_public_path__ + "path/to/asset.ext";
	// export default "path/to/asset.ext";
	// module.exports = __webpack_public_path__ + "path/to/asset.ext";
	webpackAssetRe = regexp.MustCompile(`(?:export\s+default|module\.exports\s*=)\s+(?:__webpack_public_path__\s*\+\s*)?"([^"]+\.(svg|png|jpg|jpeg|gif|webp|ico|woff|woff2|ttf|eot|otf|mp3|wav|ogg|mp4|webm))";?`)
)

// DownloadResult contains the results of a webpack asset download operation.
type DownloadResult struct {
	DownloadedCount int
	Errors          []error
}

// DownloadWebpackAssets scans restored sources for webpack asset references,
// downloads the actual assets, and replaces the fake loader files in-place.
func DownloadWebpackAssets(baseURL, inputDir string, client *fetch.Client) DownloadResult {
	result := DownloadResult{}

	// Parse base URL to construct asset URLs
	parsed, err := url.Parse(baseURL)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("invalid base URL: %w", err))
		return result
	}

	// Build origin URL (scheme + host)
	origin := fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host)

	err = filepath.WalkDir(inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("walk error at %s: %w", path, err))
			return nil
		}

		if d.IsDir() {
			return nil
		}

		downloaded, downloadErr := processWebpackAsset(path, origin, client)
		if downloadErr != nil {
			result.Errors = append(result.Errors, downloadErr)
			return nil
		}

		if downloaded {
			result.DownloadedCount++
		}

		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to walk directory: %w", err))
	}

	return result
}

// processWebpackAsset checks if a file contains a webpack asset reference,
// downloads the actual asset, and replaces the file content.
// Returns true if an asset was downloaded and replaced.
func processWebpackAsset(filePath, origin string, client *fetch.Client) (bool, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	matches := webpackAssetRe.FindSubmatch(content)
	if matches == nil {
		return false, nil
	}

	assetPath := string(matches[1])
	assetURL := origin + "/" + assetPath

	// Download the actual asset
	assetData, err := client.GetBytes(assetURL)
	if err != nil {
		return false, fmt.Errorf("failed to download asset %s: %w", assetURL, err)
	}

	// Determine correct extension from the downloaded asset path
	correctExt := filepath.Ext(assetPath)
	currentExt := filepath.Ext(filePath)

	// If extensions differ, rename the file
	newPath := filePath
	if !strings.EqualFold(correctExt, currentExt) && correctExt != "" {
		// Keep the base name but use the correct extension
		newPath = strings.TrimSuffix(filePath, currentExt) + correctExt
	}

	// Write the actual asset content
	if err := os.WriteFile(newPath, assetData, 0644); err != nil {
		return false, fmt.Errorf("failed to write asset %s: %w", newPath, err)
	}

	// Remove the old file if we renamed it
	if newPath != filePath {
		os.Remove(filePath)
	}

	return true, nil
}
