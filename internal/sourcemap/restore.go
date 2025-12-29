package sourcemap

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/thesavant42/dejank/internal/format"
)

var (
	// Characters illegal in file paths on Windows
	illegalCharsRe = regexp.MustCompile(`[<>:"|?*\x00-\x1F]`)

	// Matches webpack asset exports:
	// export default __webpack_public_path__ + "static/media/file.hash.ext"
	// export default "static/media/file.hash.ext"
	// module.exports = __webpack_public_path__ + "static/media/file.hash.ext"
	webpackAssetExportRe = regexp.MustCompile(`(?:export\s+default|module\.exports\s*=)\s+(?:__webpack_public_path__\s*\+\s*)?"([^"]+)"`)

	// Media file extensions that might be webpack loader stubs
	mediaExtensions = map[string]bool{
		".svg": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
		".webp": true, ".ico": true, ".bmp": true,
		".woff": true, ".woff2": true, ".ttf": true, ".eot": true, ".otf": true,
		".mp3": true, ".wav": true, ".ogg": true,
		".mp4": true, ".webm": true,
	}
)

// AssetFetcher can download assets from URLs
type AssetFetcher interface {
	GetBytes(url string) ([]byte, error)
}

// isMediaExtension checks if a path has a media file extension
func isMediaExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return mediaExtensions[ext]
}

// isJavaScriptContent checks if content appears to be JavaScript rather than media
func isJavaScriptContent(content string) bool {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return false
	}

	// Check for common JS patterns at the start
	jsStarters := []string{
		"import ", "import{", "import(",
		"export ", "export{",
		"module.exports",
		"var ", "let ", "const ",
		"function ", "function(",
		"//", "/*",
		"\"use strict\"", "'use strict'",
	}

	for _, starter := range jsStarters {
		if strings.HasPrefix(trimmed, starter) {
			return true
		}
	}

	return false
}

// extractWebpackAssetURL extracts the asset URL from webpack loader stub content
func extractWebpackAssetURL(content string) string {
	matches := webpackAssetExportRe.FindStringSubmatch(content)
	if len(matches) >= 2 {
		return matches[1]
	}
	return ""
}

// RestoreResult contains the result of a restore operation.
type RestoreResult struct {
	RestoredCount int
	SkippedCount  int
	AssetsFetched int
	Errors        []error
}

// RestoreOptions configures how sources are restored.
type RestoreOptions struct {
	BaseURL string       // Base URL for resolving relative asset paths
	Fetcher AssetFetcher // HTTP client for fetching real assets (nil = skip fetching)
}

// RestoreSources extracts all sources from a sourcemap to the output directory.
func RestoreSources(sm *SourceMap, outputDir string) RestoreResult {
	return RestoreSourcesWithOptions(sm, outputDir, nil)
}

// RestoreSourcesWithOptions extracts sources with optional asset fetching.
func RestoreSourcesWithOptions(sm *SourceMap, outputDir string, opts *RestoreOptions) RestoreResult {
	result := RestoreResult{}

	if len(sm.SourcesContent) == 0 {
		return result
	}

	for i, source := range sm.Sources {
		if i >= len(sm.SourcesContent) {
			break
		}

		content := sm.SourcesContent[i]
		if content == "" {
			result.SkippedCount++
			continue
		}

		virtualPath := sanitizePath(source)
		if virtualPath == "" || len(virtualPath) > 255 {
			virtualPath = fmt.Sprintf("source_%d.js", i)
		}

		outPath := filepath.Join(outputDir, virtualPath)

		// Check if this is a media file with JS stub content
		if isMediaExtension(virtualPath) && isJavaScriptContent(content) {
			if opts != nil && opts.Fetcher != nil && opts.BaseURL != "" {
				// Try to fetch the real asset
				if fetched := tryFetchRealAsset(content, outPath, opts); fetched {
					result.AssetsFetched++
					result.RestoredCount++
					continue
				}
			}
			// If we can't fetch, skip writing the stub file entirely
			result.SkippedCount++
			continue
		}

		if err := writeFile(outPath, content); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to restore %s: %w", source, err))
			continue
		}

		result.RestoredCount++
	}

	return result
}

// tryFetchRealAsset attempts to download the real asset from a webpack stub.
// Returns true if successful.
func tryFetchRealAsset(content, outPath string, opts *RestoreOptions) bool {
	assetPath := extractWebpackAssetURL(content)
	if assetPath == "" {
		return false
	}

	// Resolve the asset URL against the base URL
	assetURL, err := resolveAssetURL(opts.BaseURL, assetPath)
	if err != nil {
		return false
	}

	// Fetch the real asset
	data, err := opts.Fetcher.GetBytes(assetURL)
	if err != nil {
		return false
	}

	// Create parent directories
	dir := filepath.Dir(outPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false
	}

	// Write the real asset data
	if err := os.WriteFile(outPath, data, 0644); err != nil {
		return false
	}

	return true
}

// resolveAssetURL resolves a relative asset path against a base URL.
func resolveAssetURL(baseURL, assetPath string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Asset paths are typically relative to site root (e.g., "static/media/file.svg")
	// Construct absolute URL from base scheme + host + asset path
	resolved := &url.URL{
		Scheme: base.Scheme,
		Host:   base.Host,
		Path:   "/" + strings.TrimPrefix(assetPath, "/"),
	}

	return resolved.String(), nil
}

// sanitizePath cleans a source path for safe filesystem use.
func sanitizePath(source string) string {
	// Remove webpack:// prefix
	path := strings.TrimPrefix(source, "webpack://")

	// Remove leading ./ or multiple ./
	for strings.HasPrefix(path, "./") {
		path = strings.TrimPrefix(path, "./")
	}

	// Normalize path separators
	path = filepath.FromSlash(path)

	// Split into parts and sanitize each segment
	parts := strings.Split(path, string(filepath.Separator))
	sanitized := make([]string, 0, len(parts))

	for _, part := range parts {
		clean := sanitizePathSegment(part)
		if clean != "" {
			sanitized = append(sanitized, clean)
		}
	}

	if len(sanitized) == 0 {
		return ""
	}

	return filepath.Join(sanitized...)
}

// sanitizePathSegment cleans a single path segment.
func sanitizePathSegment(segment string) string {
	if !utf8.ValidString(segment) {
		return ""
	}

	// Remove illegal characters
	clean := illegalCharsRe.ReplaceAllString(segment, "")

	// Replace spaces with underscores
	clean = strings.ReplaceAll(clean, " ", "_")

	// Remove trailing dots
	clean = strings.TrimRight(clean, ".")

	// Remove leading dots (but preserve single dot for hidden files on Unix)
	for len(clean) > 1 && clean[0] == '.' && clean[1] == '.' {
		clean = clean[1:]
	}

	return clean
}

// writeFile writes content to a file, creating parent directories as needed.
// JS/TS files are pretty-printed before writing.
func writeFile(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Pretty-print JS/TS files (non-JS files pass through unchanged)
	formatted := format.Format(content, path)

	return os.WriteFile(path, []byte(formatted), 0644)
}
