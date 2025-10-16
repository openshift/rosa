package helper

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	humanize "github.com/dustin/go-humanize"
)

// download will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory. We pass an io.TeeReader
// into Copy() to report progress on the download.
func Download(url string, filename string) error {
	// Create a temporary file in the same directory as the target file
	// This ensures atomic rename and avoids cross-device issues
	dir := filepath.Dir(filename)
	base := filepath.Base(filename)

	// Create temp file with pattern "basename.*.tmp"
	// For example: "rosa-linux.tar.gz" becomes "rosa-linux.tar.gz.123456.tmp"
	out, err := os.CreateTemp(dir, base+".*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %v", err)
	}
	tmpFile := out.Name()

	// Ensure cleanup of temp file on any error
	cleanupTempFile := func() {
		out.Close()
		os.Remove(tmpFile)
	}

	// Get the data
	// nolint:gosec
	resp, err := http.Get(url)
	if err != nil {
		cleanupTempFile()
		return formatDownloadError(err, url)
	}
	defer resp.Body.Close()

	// Check for 2xx success status codes
	if resp.StatusCode/100 != 2 {
		cleanupTempFile()
		switch resp.StatusCode {
		case http.StatusNotFound:
			return fmt.Errorf("download failed: file not found (HTTP %d). The requested file may not exist or the URL may be incorrect. URL: %s", resp.StatusCode, url)
		case http.StatusForbidden:
			return fmt.Errorf("download failed: access forbidden (HTTP %d). You may not have permission to access this file. URL: %s", resp.StatusCode, url)
		case http.StatusUnauthorized:
			return fmt.Errorf("download failed: authentication required (HTTP %d). Please check your credentials. URL: %s", resp.StatusCode, url)
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable:
			return fmt.Errorf("download failed: server error (HTTP %d). The server may be temporarily unavailable. Please try again later. URL: %s", resp.StatusCode, url)
		default:
			return fmt.Errorf("download failed: HTTP %d %s. URL: %s", resp.StatusCode, resp.Status, url)
		}
	}

	// Create our progress reporter and pass it to be used alongside our writer
	counter := &WriteCounter{}
	if _, err = io.Copy(out, io.TeeReader(resp.Body, counter)); err != nil {
		cleanupTempFile()
		return fmt.Errorf("failed to save downloaded file: %v", err)
	}

	// The progress use the same line so print a new line once it's finished downloading
	fmt.Print("\n")

	// Close the file without defer so it can happen before Rename()
	out.Close()

	if err = os.Rename(tmpFile, filename); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to rename downloaded file: %v", err)
	}
	return nil
}

// formatDownloadError formats network and download errors in a user-friendly way
func formatDownloadError(err error, url string) error {
	if err == nil {
		return nil
	}

	// Check for common network error types
	errStr := err.Error()

	// DNS resolution errors
	if strings.Contains(errStr, "lookup") || strings.Contains(errStr, "no such host") {
		return fmt.Errorf("unable to resolve host for %s\nPlease check your internet connection and DNS settings", url)
	}

	// Connection refused
	if strings.Contains(errStr, "connection refused") {
		return fmt.Errorf("connection refused to %s\nThe server may be down or unreachable", url)
	}

	// Timeout errors
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "i/o timeout") {
		return fmt.Errorf("connection timeout while downloading from %s\nPlease check your internet connection and try again", url)
	}

	// Network unreachable
	if strings.Contains(errStr, "network is unreachable") {
		return fmt.Errorf("network is unreachable for %s\nPlease check your internet connection", url)
	}

	// Certificate errors
	if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "x509") {
		return fmt.Errorf("certificate validation failed for %s\nThe server's SSL certificate may be invalid or expired", url)
	}

	// Check if it's a net.OpError for more specific handling
	if _, ok := err.(*net.OpError); ok {
		// Generic network operation error - provide a clean message
		return fmt.Errorf("network error while downloading from %s\nPlease check your internet connection and try again", url)
	}

	// Default case - still provide a cleaner message than raw error
	return fmt.Errorf("download failed: unable to connect to %s\nPlease check your internet connection and try again", url)
}

// Get the extension used for the compressed oc file
func GetExtension() string {
	if runtime.GOOS == "windows" {
		return "zip"
	}
	return "tar.gz"
}

// WriteCounter counts the number of bytes written to it. It implements to the io.Writer interface
// and we can pass this into io.TeeReader() which will report progress on each write cycle.
type WriteCounter struct {
	Total uint64
}

func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}

func (wc WriteCounter) PrintProgress() {
	// Clear the line by using a character return to go back to the start and remove
	// the remaining characters by filling it with spaces
	fmt.Printf("\r%s", strings.Repeat(" ", 35))

	// Return again and print current status of download
	// We use the humanize package to print the bytes in a meaningful way (e.g. 10 MB)
	fmt.Printf("\rDownloading... %s complete", humanize.Bytes(wc.Total))
}
