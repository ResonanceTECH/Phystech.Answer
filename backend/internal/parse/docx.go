package parse

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
)

// extractDOCX pulls paragraph text from word/document.xml inside the OOXML
// zip container. Paragraph boundaries are preserved as newlines.
func extractDOCX(data []byte) (string, error) {
	zr, err := openZip(data)
	if err != nil {
		return "", fmt.Errorf("docx: not a zip: %w", err)
	}
	var docFile io.ReadCloser
	for _, f := range zr.File {
		if f.Name == "word/document.xml" {
			docFile, err = f.Open()
			if err != nil {
				return "", fmt.Errorf("docx: open document.xml: %w", err)
			}
			break
		}
	}
	if docFile == nil {
		return "", fmt.Errorf("docx: word/document.xml missing")
	}
	defer docFile.Close()

	var b strings.Builder
	dec := xml.NewDecoder(docFile)
	inParagraph := false
	for {
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("docx: parse xml: %w", err)
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "p":
				inParagraph = true
			case "tab":
				if inParagraph {
					b.WriteByte('\t')
				}
			case "br":
				b.WriteByte('\n')
			}
		case xml.EndElement:
			if t.Name.Local == "p" && inParagraph {
				inParagraph = false
				b.WriteByte('\n')
			}
		case xml.CharData:
			if inParagraph {
				b.Write(t)
			}
		}
	}
	text := strings.TrimSpace(cleanText(b.String()))
	if text == "" {
		return "", ErrNoText
	}
	return text, nil
}
