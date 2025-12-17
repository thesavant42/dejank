// Package format provides pretty-printing for JavaScript/TypeScript source files.
package format

import (
	"path/filepath"
	"strings"

	"github.com/ditashi/jsbeautifier-go/jsbeautifier"
)

// isJSFile returns true if the file extension indicates a JS/TS file.
func isJSFile(ext string) bool {
	switch strings.ToLower(ext) {
	case ".js", ".mjs", ".cjs", ".jsx", ".ts", ".mts", ".cts", ".tsx":
		return true
	default:
		return false
	}
}

// Format pretty-prints JavaScript/TypeScript content using jsbeautifier.
// Returns the formatted content, or the original content if formatting fails
// or the file type is not supported.
func Format(content string, filename string) string {
	ext := filepath.Ext(filename)

	// Not a JS/TS file, return unchanged
	if !isJSFile(ext) {
		return content
	}

	options := jsbeautifier.DefaultOptions()
	result, err := jsbeautifier.Beautify(&content, options)
	if err != nil {
		// If beautification fails, return original content (graceful fallback)
		return content
	}

	return result
}
