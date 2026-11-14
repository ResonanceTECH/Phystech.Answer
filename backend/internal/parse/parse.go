// Package parse extracts text and structure from uploaded regulation files.
// All extraction is pure-Go with stdlib only; files that yield no usable
// text (e.g. scanned PDFs without a text layer) return ErrNoText so the
// caller can route them to requires_review / OCR.
package parse

import (
	"archive/zip"
	"bytes"
	"fmt"
	"path"
	"strings"
)

// ErrNoText marks files from which no usable text could be extracted.
var ErrNoText = fmt.Errorf("no extractable text")

// FileType is the detected container format.
type FileType string

const (
	TypeTXT   FileType = "txt"
	TypeDOCX  FileType = "docx"
	TypeXLSX  FileType = "xlsx"
	TypePDF   FileType = "pdf"
	TypeOther FileType = "other"
)

// DetectByName maps a filename extension to a FileType.
func DetectByName(filename string) FileType {
	switch strings.ToLower(path.Ext(filename)) {
	case ".txt", ".md":
		return TypeTXT
	case ".docx":
		return TypeDOCX
	case ".xlsx":
		return TypeXLSX
	case ".pdf":
		return TypePDF
	default:
		return TypeOther
	}
}

// Supported reports whether Extract can handle this file type.
func Supported(t FileType) bool {
	return t != TypeOther
}

// Extract returns plain text content of the file.
func Extract(t FileType, data []byte) (string, error) {
	switch t {
	case TypeTXT:
		return cleanText(string(data)), nil
	case TypeDOCX:
		return extractDOCX(data)
	case TypeXLSX:
		return extractXLSX(data)
	case TypePDF:
		return extractPDF(data)
	default:
		return "", fmt.Errorf("unsupported file type %q", t)
	}
}

func openZip(data []byte) (*zip.Reader, error) {
	return zip.NewReader(bytes.NewReader(data), int64(len(data)))
}

// cleanText normalizes line endings and strips NULs.
func cleanText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.Map(func(r rune) rune {
		if r == 0 {
			return -1
		}
		return r
	}, s)
}
