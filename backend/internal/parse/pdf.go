package parse

import (
	"bytes"
	"compress/flate"
	"compress/zlib"
	"io"
	"regexp"
	"strings"
)

// extractPDF performs best-effort text extraction from PDFs that carry a
// text layer: it inflates content streams and collects Tj/TJ string
// operands. Scanned PDFs (image-only) yield no text and surface as
// ErrNoText so the pipeline can route them to OCR / requires_review.
func extractPDF(data []byte) (string, error) {
	var b strings.Builder
	for _, stream := range pdfStreams(data) {
		extractStreamText(stream, &b)
	}
	text := strings.TrimSpace(cleanText(b.String()))
	if text == "" {
		return "", ErrNoText
	}
	return text, nil
}

var streamRe = regexp.MustCompile(`(?s)stream\r?\n(.*?)\r?\n?endstream`)

// pdfStreams returns decoded contents of every stream object. Encrypted or
// non-Flate streams are skipped silently.
func pdfStreams(data []byte) [][]byte {
	var out [][]byte
	for _, m := range streamRe.FindAllSubmatch(data, -1) {
		raw := m[1]
		if decoded := inflate(raw); len(decoded) > 0 {
			out = append(out, decoded)
			continue
		}
		// Some PDFs store uncompressed text streams.
		if bytes.Contains(raw, []byte("Tj")) || bytes.Contains(raw, []byte("TJ")) {
			out = append(out, raw)
		}
	}
	return out
}

// inflate tries zlib (PDF FlateDecode) first, then raw DEFLATE.
func inflate(raw []byte) []byte {
	if r, err := zlib.NewReader(bytes.NewReader(raw)); err == nil {
		if dec, err := io.ReadAll(r); err == nil && len(dec) > 0 {
			r.Close()
			return dec
		}
		r.Close()
	}
	if r := flate.NewReader(bytes.NewReader(raw)); r != nil {
		if dec, err := io.ReadAll(r); err == nil && len(dec) > 0 {
			r.Close()
			return dec
		}
		r.Close()
	}
	return nil
}

var (
	pdfStringRe = regexp.MustCompile(`\((?:\\.|[^\\()])*\)`)
	pdfEscapeRe = regexp.MustCompile(`\\(n|r|t|b|f|\(|\)|\\|[0-7]{1,3})`)
)

// extractStreamText pulls text-showing operators in order: (...)Tj,
// [...]TJ, ' and " variants. Each Tj/TJ array emits a line break.
func extractStreamText(stream []byte, b *strings.Builder) {
	for _, m := range pdfStringRe.FindAll(stream, -1) {
		s := unescapePDFString(string(m[1 : len(m)-1]))
		if strings.TrimSpace(s) == "" {
			continue
		}
		b.WriteString(s)
		b.WriteByte('\n')
	}
}

func unescapePDFString(s string) string {
	return pdfEscapeRe.ReplaceAllStringFunc(s, func(e string) string {
		switch e[1] {
		case 'n':
			return "\n"
		case 'r':
			return "\r"
		case 't':
			return "\t"
		case '(', ')', '\\':
			return string(e[1])
		default:
			// Octal escape \ddd.
			v := 0
			for _, c := range e[1:] {
				v = v*8 + int(c-'0')
			}
			return string(rune(v))
		}
	})
}
