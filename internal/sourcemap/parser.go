package sourcemap

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	// Matches //# sourceMappingURL=... or //@ sourceMappingURL=...
	sourceMappingURLRe = regexp.MustCompile(`//[#@]\s*sourceMappingURL\s*=\s*([^\s]+)`)

	// Matches inline base64 sourcemaps
	inlineSourceMapRe = regexp.MustCompile(`sourceMappingURL\s*=\s*data:application/json[^,]*;base64,([a-zA-Z0-9+/=]+)`)
)

// ParseFile reads and parses a sourcemap from a file path.
func ParseFile(path string) (*SourceMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read sourcemap file: %w", err)
	}

	return Parse(data)
}

// Parse parses sourcemap JSON data.
func Parse(data []byte) (*SourceMap, error) {
	var sm SourceMap
	if err := json.Unmarshal(data, &sm); err != nil {
		return nil, fmt.Errorf("failed to parse sourcemap JSON: %w", err)
	}

	return &sm, nil
}

// ExtractSourceMappingURL finds the sourceMappingURL comment in JS content.
// Returns empty string if not found or if it's an inline data URI.
func ExtractSourceMappingURL(jsContent string) string {
	// Search from the end of the file (more efficient for large bundles)
	lines := strings.Split(strings.TrimSpace(jsContent), "\n")

	// Check last 10 lines (sourcemap comment is typically at the very end)
	start := len(lines) - 10
	if start < 0 {
		start = 0
	}

	for i := len(lines) - 1; i >= start; i-- {
		line := lines[i]
		matches := sourceMappingURLRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			url := strings.TrimSpace(matches[1])
			// Skip data URIs - those are handled by ExtractInlineSourceMap
			if strings.HasPrefix(url, "data:") {
				return ""
			}
			return url
		}
	}

	return ""
}

// ExtractInlineSourceMap extracts and decodes a base64 inline sourcemap from JS content.
// Returns nil if no inline sourcemap is found.
func ExtractInlineSourceMap(jsContent string) (*SourceMap, error) {
	// Search from the end
	lines := strings.Split(strings.TrimSpace(jsContent), "\n")

	start := len(lines) - 10
	if start < 0 {
		start = 0
	}

	for i := len(lines) - 1; i >= start; i-- {
		line := lines[i]
		if !strings.Contains(line, "sourceMappingURL=data:application/json") {
			continue
		}

		matches := inlineSourceMapRe.FindStringSubmatch(line)
		if len(matches) >= 2 {
			decoded, err := base64.StdEncoding.DecodeString(matches[1])
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 sourcemap: %w", err)
			}

			return Parse(decoded)
		}
	}

	return nil, nil
}

// HasInlineSourceMap checks if JS content contains an inline sourcemap.
func HasInlineSourceMap(jsContent string) bool {
	return strings.Contains(jsContent, "sourceMappingURL=data:application/json")
}

