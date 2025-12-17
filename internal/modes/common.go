// Package modes implements the three operation modes: url, single, and local.
package modes

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/thesavant42/dejank/internal/fetch"
)

// ProgressCallback is called to report progress during operations.
type ProgressCallback func(event string, data interface{})

// Config holds configuration for all modes.
type Config struct {
	OutputRoot string       // Root output directory (default: .)
	Client     *fetch.Client
	Verbose    bool
	Force      bool         // Overwrite existing output directory
	OnProgress ProgressCallback // Optional callback for progress events
}

// emit sends a progress event if a callback is configured.
func (c *Config) emit(event string, data interface{}) {
	if c.OnProgress != nil {
		c.OnProgress(event, data)
	}
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		OutputRoot: ".",
		Client:     fetch.New(),
		Verbose:    false,
	}
}

// DomainPaths holds the standard directory structure for a domain.
type DomainPaths struct {
	Base           string // output/<domain>
	DownloadedSite string // output/<domain>/downloaded_site
	RestoredSources string // output/<domain>/restored_sources
	ExtractedAssets string // output/<domain>/extracted_assets
}

// GetDomainPaths returns the standard directory paths for a domain.
func GetDomainPaths(outputRoot, domain string) DomainPaths {
	base := filepath.Join(outputRoot, sanitizeDomain(domain))
	return DomainPaths{
		Base:           base,
		DownloadedSite: filepath.Join(base, "downloaded_site"),
		RestoredSources: filepath.Join(base, "restored_sources"),
		ExtractedAssets: filepath.Join(base, "extracted_assets"),
	}
}

// EnsureDirs creates all directories in a DomainPaths struct.
func (dp DomainPaths) EnsureDirs() error {
	dirs := []string{dp.DownloadedSite, dp.RestoredSources, dp.ExtractedAssets}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}
	return nil
}

// Exists checks if the base domain directory already exists.
func (dp DomainPaths) Exists() bool {
	_, err := os.Stat(dp.Base)
	return err == nil
}

// sanitizeDomain cleans a domain name for use as a directory name.
func sanitizeDomain(domain string) string {
	// Remove port if present
	if idx := strings.LastIndex(domain, ":"); idx != -1 {
		domain = domain[:idx]
	}
	return domain + "-dejank"
}

// resolveURL resolves a potentially relative URL against a base URL.
func resolveURL(baseURL, ref string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	refURL, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("invalid reference URL: %w", err)
	}

	return base.ResolveReference(refURL).String(), nil
}

// filenameFromURL extracts a clean filename from a URL.
func filenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return "unknown.js"
	}

	// Get path basename, strip query params
	base := filepath.Base(parsed.Path)
	if base == "" || base == "/" || base == "." {
		return "index.js"
	}

	return base
}

