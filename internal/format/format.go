// Package format provides pretty-printing for JavaScript/TypeScript source files.
package format

import (
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// loaderForExt returns the appropriate esbuild loader for a file extension.
// Returns LoaderNone if the extension is not a supported JS/TS type.
func loaderForExt(ext string) api.Loader {
	switch strings.ToLower(ext) {
	case ".js", ".mjs", ".cjs":
		return api.LoaderJS
	case ".jsx":
		return api.LoaderJSX
	case ".ts", ".mts", ".cts":
		return api.LoaderTS
	case ".tsx":
		return api.LoaderTSX
	default:
		return api.LoaderNone
	}
}

// Format pretty-prints JavaScript/TypeScript content using esbuild.
// Returns the formatted content, or the original content if formatting fails
// or the file type is not supported.
func Format(content string, filename string) string {
	ext := filepath.Ext(filename)
	loader := loaderForExt(ext)

	// Not a JS/TS file, return unchanged
	if loader == api.LoaderNone {
		return content
	}

	result := api.Transform(content, api.TransformOptions{
		Loader: loader,
		// No minification - produce readable output
		MinifySyntax:      false,
		MinifyWhitespace:  false,
		MinifyIdentifiers: false,
		// Preserve original structure as much as possible
		KeepNames: true,
	})

	// If there are errors, return original content (graceful fallback)
	if len(result.Errors) > 0 {
		return content
	}

	return string(result.Code)
}

