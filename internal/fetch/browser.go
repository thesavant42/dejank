// Package fetch provides browser-based JavaScript execution for discovering scripts.
package fetch

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
)

// DiscoveredResources contains all JS and sourcemap URLs found during page load.
type DiscoveredResources struct {
	Scripts    []string // All .js URLs loaded
	SourceMaps []string // All .map URLs loaded
	BaseURL    string   // The final URL after redirects
}

// BrowserClient uses headless Chrome to execute JavaScript and discover resources.
type BrowserClient struct {
	timeout time.Duration
}

// NewBrowserClient creates a new browser-based client.
func NewBrowserClient() *BrowserClient {
	return &BrowserClient{
		timeout: 60 * time.Second,
	}
}

// DiscoverResources loads a URL in headless Chrome, executes all JavaScript,
// and returns all discovered script and sourcemap URLs.
func (b *BrowserClient) DiscoverResources(targetURL string) (*DiscoveredResources, error) {
	// Suppress chromedp's noisy error logging for unknown CDP values
	log.SetOutput(io.Discard)
	defer log.SetOutput(log.Writer())

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), b.timeout)
	defer cancel()

	// Create Chrome options
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("ignore-certificate-errors", true),
	)

	allocCtx, allocCancel := chromedp.NewExecAllocator(ctx, opts...)
	defer allocCancel()

	browserCtx, browserCancel := chromedp.NewContext(allocCtx, chromedp.WithLogf(func(string, ...interface{}) {}))
	defer browserCancel()

	result := &DiscoveredResources{
		Scripts:    make([]string, 0),
		SourceMaps: make([]string, 0),
	}

	var mu sync.Mutex
	seen := make(map[string]bool)

	// Enable network events and listen for requests
	chromedp.ListenTarget(browserCtx, func(ev interface{}) {
		switch e := ev.(type) {
		case *network.EventRequestWillBeSent:
			reqURL := e.Request.URL
			mu.Lock()
			defer mu.Unlock()

			if seen[reqURL] {
				return
			}
			seen[reqURL] = true

			// Check for JS files
			if isJavaScriptURL(reqURL) {
				result.Scripts = append(result.Scripts, reqURL)
			}

			// Check for sourcemap files
			if isSourceMapURL(reqURL) {
				result.SourceMaps = append(result.SourceMaps, reqURL)
			}

		case *network.EventResponseReceived:
			// Check for sourcemap headers
			if e.Response != nil && e.Response.Headers != nil {
				if smURL, ok := e.Response.Headers["SourceMap"]; ok {
					if smStr, ok := smURL.(string); ok && smStr != "" {
						mu.Lock()
						if !seen[smStr] {
							seen[smStr] = true
							resolved := resolveMapURL(e.Response.URL, smStr)
							result.SourceMaps = append(result.SourceMaps, resolved)
						}
						mu.Unlock()
					}
				}
				// Also check X-SourceMap header (older convention)
				if smURL, ok := e.Response.Headers["X-SourceMap"]; ok {
					if smStr, ok := smURL.(string); ok && smStr != "" {
						mu.Lock()
						if !seen[smStr] {
							seen[smStr] = true
							resolved := resolveMapURL(e.Response.URL, smStr)
							result.SourceMaps = append(result.SourceMaps, resolved)
						}
						mu.Unlock()
					}
				}
			}
		}
	})

	// Navigate and wait for page to be fully loaded
	var finalURL string
	err := chromedp.Run(browserCtx,
		network.Enable(),
		chromedp.Navigate(targetURL),
		chromedp.WaitReady("body"),
		// Wait for network to settle - longer wait for SPAs that lazy-load
		chromedp.Sleep(5*time.Second),
		chromedp.Location(&finalURL),
	)

	if err != nil {
		return nil, fmt.Errorf("browser navigation failed: %w", err)
	}

	result.BaseURL = finalURL

	return result, nil
}

// isJavaScriptURL checks if a URL points to a JavaScript file.
func isJavaScriptURL(u string) bool {
	// Parse URL to get path without query params
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	path := strings.ToLower(parsed.Path)

	// Check extension
	if strings.HasSuffix(path, ".js") || strings.HasSuffix(path, ".mjs") {
		return true
	}

	// Check for webpack chunks and similar patterns
	if strings.Contains(path, "/chunk") && strings.Contains(u, ".js") {
		return true
	}

	return false
}

// isSourceMapURL checks if a URL points to a sourcemap file.
func isSourceMapURL(u string) bool {
	parsed, err := url.Parse(u)
	if err != nil {
		return false
	}

	path := strings.ToLower(parsed.Path)
	return strings.HasSuffix(path, ".map") || strings.HasSuffix(path, ".js.map")
}

// resolveMapURL resolves a potentially relative sourcemap URL against a base URL.
func resolveMapURL(baseURL, mapURL string) string {
	if strings.HasPrefix(mapURL, "http://") || strings.HasPrefix(mapURL, "https://") {
		return mapURL
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return mapURL
	}

	ref, err := url.Parse(mapURL)
	if err != nil {
		return mapURL
	}

	return base.ResolveReference(ref).String()
}

