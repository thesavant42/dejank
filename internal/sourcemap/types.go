// Package sourcemap handles parsing and restoring sources from JavaScript sourcemaps.
package sourcemap

// SourceMap represents a JavaScript sourcemap structure.
type SourceMap struct {
	Version        int      `json:"version"`
	File           string   `json:"file,omitempty"`
	SourceRoot     string   `json:"sourceRoot,omitempty"`
	Sources        []string `json:"sources"`
	SourcesContent []string `json:"sourcesContent,omitempty"`
	Names          []string `json:"names,omitempty"`
	Mappings       string   `json:"mappings,omitempty"`

	// Non-standard fields for toolchain detection
	XFacebookSources  interface{} `json:"x_facebook_sources,omitempty"`
	XGoogleIgnoreList interface{} `json:"x_google_ignoreList,omitempty"`
	Sections          []struct{}  `json:"sections,omitempty"`
}

// Metadata contains summary information about a sourcemap.
type Metadata struct {
	File              string
	Version           int
	SourceCount       int
	HasSourcesContent bool
	NamesCount        int
	HasMappings       bool
	SourceRoot        string
	SectionCount      int
	ToolchainHints    []string
}

// ExtractMetadata extracts summary metadata from a SourceMap.
func (sm *SourceMap) ExtractMetadata() Metadata {
	meta := Metadata{
		File:              sm.File,
		Version:           sm.Version,
		SourceCount:       len(sm.Sources),
		HasSourcesContent: len(sm.SourcesContent) > 0,
		NamesCount:        len(sm.Names),
		HasMappings:       len(sm.Mappings) > 0,
		SourceRoot:        sm.SourceRoot,
		SectionCount:      len(sm.Sections),
		ToolchainHints:    []string{},
	}

	// Detect toolchain hints
	if sm.XFacebookSources != nil {
		meta.ToolchainHints = append(meta.ToolchainHints, "Facebook (Metro bundler)")
	}
	if sm.XGoogleIgnoreList != nil {
		meta.ToolchainHints = append(meta.ToolchainHints, "Chrome DevTools")
	}
	if containsWebpack(sm.SourceRoot) {
		meta.ToolchainHints = append(meta.ToolchainHints, "Webpack")
	}
	for _, src := range sm.Sources {
		if len(src) > 10 && src[:10] == "webpack://" {
			meta.ToolchainHints = append(meta.ToolchainHints, "Webpack")
			break
		}
	}

	return meta
}

func containsWebpack(s string) bool {
	return len(s) >= 7 && (s == "webpack" || (len(s) > 7 && s[:8] == "webpack:"))
}

