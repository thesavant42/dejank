// Package assets handles extraction of base64-encoded assets from restored source files.
package assets

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	// Matches: export default "data:image/png;base64,..."
	base64ExportRe = regexp.MustCompile(`^export default\s+"data:(.+?);base64,(.+)";?$`)
)

// MIME type to file extension mapping.
var mimeToExt = map[string]string{
	"image/png":                    "png",
	"image/jpeg":                   "jpg",
	"image/gif":                    "gif",
	"image/svg+xml":                "svg",
	"image/webp":                   "webp",
	"font/woff":                    "woff",
	"font/woff2":                   "woff2",
	"font/ttf":                     "ttf",
	"font/otf":                     "otf",
	"application/vnd.ms-fontobject": "eot",
	"application/font-sfnt":        "sfnt",
	"application/font-woff":        "woff",
	"application/font-woff2":       "woff2",
	"application/octet-stream":     "bin",
}

// ExtractResult contains the results of an extraction operation.
type ExtractResult struct {
	ExtractedCount int
	Errors         []error
}

// ExtractFromDirectory walks a directory and extracts base64 assets from all files.
func ExtractFromDirectory(inputDir, outputDir string) ExtractResult {
	result := ExtractResult{}

	err := filepath.WalkDir(inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("walk error at %s: %w", path, err))
			return nil // Continue walking
		}

		if d.IsDir() {
			return nil
		}

		extracted, err := ExtractFromFile(path, outputDir)
		if err != nil {
			result.Errors = append(result.Errors, err)
			return nil
		}

		if extracted != "" {
			result.ExtractedCount++
		}

		return nil
	})

	if err != nil {
		result.Errors = append(result.Errors, fmt.Errorf("failed to walk directory: %w", err))
	}

	return result
}

// ExtractFromFile checks if a file contains a base64 export and extracts it.
// Returns the output path if extracted, empty string otherwise.
func ExtractFromFile(filePath, outputDir string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	text := strings.TrimSpace(string(content))
	matches := base64ExportRe.FindStringSubmatch(text)
	if matches == nil {
		return "", nil // Not a base64 export file
	}

	mime := matches[1]
	b64Data := matches[2]

	// Decode base64
	decoded, err := base64.StdEncoding.DecodeString(b64Data)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64 in %s: %w", filePath, err)
	}

	// Determine extension
	ext := extensionFromMIME(mime)

	// Build output filename
	baseName := filepath.Base(filePath)
	cleanBase := stripAllExtensions(baseName)
	outputName := cleanBase + "." + ext
	outputPath := filepath.Join(outputDir, outputName)

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write decoded file
	if err := os.WriteFile(outputPath, decoded, 0644); err != nil {
		return "", fmt.Errorf("failed to write extracted asset: %w", err)
	}

	return outputPath, nil
}

// extensionFromMIME returns the file extension for a MIME type.
func extensionFromMIME(mime string) string {
	if ext, ok := mimeToExt[mime]; ok {
		return ext
	}

	// Fallback: extract from MIME type (e.g., "image/png" -> "png")
	parts := strings.Split(mime, "/")
	if len(parts) == 2 {
		subtype := parts[1]
		// Handle cases like "svg+xml" -> "svg"
		if idx := strings.Index(subtype, "+"); idx != -1 {
			return subtype[:idx]
		}
		return subtype
	}

	return "bin"
}

// stripAllExtensions removes all extensions from a filename.
// "image.png.js" -> "image"
func stripAllExtensions(filename string) string {
	for {
		ext := filepath.Ext(filename)
		if ext == "" {
			break
		}
		filename = strings.TrimSuffix(filename, ext)
	}
	if filename == "" {
		return "asset"
	}
	return filename
}

