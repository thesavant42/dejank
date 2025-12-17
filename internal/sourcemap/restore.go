package sourcemap

import (
	"fmt"
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
)

// RestoreResult contains the result of a restore operation.
type RestoreResult struct {
	RestoredCount int
	SkippedCount  int
	Errors        []error
}

// RestoreSources extracts all sources from a sourcemap to the output directory.
func RestoreSources(sm *SourceMap, outputDir string) RestoreResult {
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

		if err := writeFile(outPath, content); err != nil {
			result.Errors = append(result.Errors, fmt.Errorf("failed to restore %s: %w", source, err))
			continue
		}

		result.RestoredCount++
	}

	return result
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

