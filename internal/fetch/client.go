// Package fetch provides HTTP utilities with insecure TLS support.
package fetch

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// Client wraps http.Client with insecure TLS configuration.
type Client struct {
	http *http.Client
}

// New creates a new Client with insecure TLS (ignores cert errors).
func New() *Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	return &Client{
		http: &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		},
	}
}

// Get fetches a URL and returns the response body as a string.
func (c *Client) Get(url string) (string, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d when fetching %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	return string(body), nil
}

// GetBytes fetches a URL and returns the response body as bytes.
func (c *Client) GetBytes(url string) ([]byte, error) {
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d when fetching %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}

// Download fetches a URL and saves it to the specified file path.
// Creates parent directories as needed.
func (c *Client) Download(url, destPath string) error {
	resp, err := c.http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d when fetching %s", resp.StatusCode, url)
	}

	// Ensure parent directory exists
	dir := filepath.Dir(destPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	file, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", destPath, err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		os.Remove(destPath) // Clean up partial file
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	return nil
}

