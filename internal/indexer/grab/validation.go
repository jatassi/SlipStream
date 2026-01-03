package grab

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrInvalidTorrent   = errors.New("invalid torrent file")
	ErrInvalidNZB       = errors.New("invalid NZB file")
	ErrMagnetLink       = errors.New("content is a magnet link, not a torrent file")
	ErrHTMLResponse     = errors.New("received HTML instead of torrent/nzb")
	ErrEmptyContent     = errors.New("empty content")
	ErrTooSmall         = errors.New("content too small to be valid")
)

// ValidateTorrent checks if the content is a valid torrent file.
// Returns nil if valid, or an error describing the issue.
func ValidateTorrent(content []byte) error {
	if len(content) == 0 {
		return ErrEmptyContent
	}

	// Minimum size for a valid torrent (bencoded dict with info)
	if len(content) < 50 {
		return ErrTooSmall
	}

	// Check for magnet link disguised as file
	if bytes.HasPrefix(content, []byte("magnet:")) {
		return ErrMagnetLink
	}

	// Check for HTML response (common error page)
	if isHTMLContent(content) {
		return ErrHTMLResponse
	}

	// Torrent files are bencoded and must start with 'd' (dictionary)
	if content[0] != 'd' {
		return fmt.Errorf("%w: does not start with bencoded dictionary", ErrInvalidTorrent)
	}

	// Check for required keys in the torrent
	// A valid torrent should contain "info" key
	if !bytes.Contains(content, []byte("4:info")) {
		return fmt.Errorf("%w: missing info dictionary", ErrInvalidTorrent)
	}

	// Verify basic bencode structure - should end with 'e'
	if content[len(content)-1] != 'e' {
		return fmt.Errorf("%w: invalid bencode structure", ErrInvalidTorrent)
	}

	return nil
}

// ValidateNZB checks if the content is a valid NZB file.
// Returns nil if valid, or an error describing the issue.
func ValidateNZB(content []byte) error {
	if len(content) == 0 {
		return ErrEmptyContent
	}

	// Minimum size for a valid NZB
	if len(content) < 100 {
		return ErrTooSmall
	}

	// Check for HTML response (common error page)
	if isHTMLContent(content) {
		return ErrHTMLResponse
	}

	// NZB files are XML and should contain the nzb namespace or root element
	contentStr := string(content)

	// Check for XML declaration or nzb root element
	if !strings.Contains(contentStr, "<?xml") && !strings.Contains(contentStr, "<nzb") {
		return fmt.Errorf("%w: not an XML document", ErrInvalidNZB)
	}

	// Check for nzb root element
	if !strings.Contains(contentStr, "<nzb") {
		return fmt.Errorf("%w: missing nzb root element", ErrInvalidNZB)
	}

	// Try to parse as XML to verify structure
	var nzb nzbDocument
	if err := xml.Unmarshal(content, &nzb); err != nil {
		return fmt.Errorf("%w: XML parse error: %v", ErrInvalidNZB, err)
	}

	// Check for required elements
	if len(nzb.Files) == 0 {
		return fmt.Errorf("%w: no files in NZB", ErrInvalidNZB)
	}

	// Verify at least one file has segments
	hasSegments := false
	for _, file := range nzb.Files {
		if len(file.Segments) > 0 {
			hasSegments = true
			break
		}
	}

	if !hasSegments {
		return fmt.Errorf("%w: no segments in NZB files", ErrInvalidNZB)
	}

	return nil
}

// ExtractMagnetURL extracts the magnet URL if content is a magnet link.
func ExtractMagnetURL(content []byte) (string, error) {
	if !bytes.HasPrefix(content, []byte("magnet:")) {
		return "", errors.New("not a magnet link")
	}

	// Find the end of the magnet link (newline or end of content)
	magnetURL := string(content)
	if idx := strings.IndexAny(magnetURL, "\r\n"); idx != -1 {
		magnetURL = magnetURL[:idx]
	}

	return strings.TrimSpace(magnetURL), nil
}

// ExtractInfoHash extracts the info hash from a torrent file.
// This is a simplified implementation that looks for the info hash pattern.
func ExtractInfoHash(content []byte) (string, error) {
	if err := ValidateTorrent(content); err != nil {
		return "", err
	}

	// Find the info dictionary
	infoIdx := bytes.Index(content, []byte("4:info"))
	if infoIdx == -1 {
		return "", fmt.Errorf("info dictionary not found")
	}

	// The actual info hash calculation requires SHA1 of the info dict
	// This would require proper bencode parsing
	// For now, return empty - a full implementation would use a bencode library
	return "", nil
}

// isHTMLContent checks if the content appears to be HTML.
func isHTMLContent(content []byte) bool {
	// Check first 1024 bytes for HTML indicators
	checkLen := len(content)
	if checkLen > 1024 {
		checkLen = 1024
	}

	check := strings.ToLower(string(content[:checkLen]))

	htmlIndicators := []string{
		"<!doctype html",
		"<html",
		"<head",
		"<body",
		"<title",
	}

	for _, indicator := range htmlIndicators {
		if strings.Contains(check, indicator) {
			return true
		}
	}

	return false
}

// NZB XML structures for validation

type nzbDocument struct {
	XMLName xml.Name  `xml:"nzb"`
	Files   []nzbFile `xml:"file"`
}

type nzbFile struct {
	Poster   string       `xml:"poster,attr"`
	Date     string       `xml:"date,attr"`
	Subject  string       `xml:"subject,attr"`
	Groups   []string     `xml:"groups>group"`
	Segments []nzbSegment `xml:"segments>segment"`
}

type nzbSegment struct {
	Bytes  int64  `xml:"bytes,attr"`
	Number int    `xml:"number,attr"`
	ID     string `xml:",chardata"`
}

// GetContentType determines the content type from the raw bytes.
func GetContentType(content []byte) string {
	if len(content) == 0 {
		return "unknown"
	}

	// Check for torrent (bencoded)
	if content[0] == 'd' && bytes.Contains(content, []byte("4:info")) {
		return "torrent"
	}

	// Check for NZB (XML)
	if bytes.Contains(content[:min(len(content), 500)], []byte("<nzb")) {
		return "nzb"
	}

	// Check for magnet link
	if bytes.HasPrefix(content, []byte("magnet:")) {
		return "magnet"
	}

	// Check for HTML
	if isHTMLContent(content) {
		return "html"
	}

	return "unknown"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
