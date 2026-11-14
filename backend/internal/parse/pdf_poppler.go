package parse

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// PDF extraction pipeline:
//  1. poppler `pdftotext` page by page (text layer, correct ToUnicode/CID
//     handling for Cyrillic);
//  2. pages with no usable text layer (scans) are rasterised with
//     `pdftoppm` and recognised by `tesseract -l rus+eng`;
//  3. if poppler is not installed, fall back to the pure-Go stream parser.
//
// Every page's text is validated: garbage (binary noise, broken encodings)
// is detected by letter ratio and routed to OCR as well.

const (
	minPageChars = 80  // fewer meaningful letters → page has no text layer
	ocrIfBelow   = 900 // thin text layer → also try OCR, keep the richer text
	ocrDPI       = "300"
	pageTimeout  = 90 * time.Second
)

// PDFStats describes how a PDF was extracted (shown in processing logs).
type PDFStats struct {
	Pages    int
	OCRPages int
	Engine   string // "poppler", "poppler+ocr", "go-fallback"
}

var popplerAvailable = func() bool {
	_, err := exec.LookPath("pdftotext")
	return err == nil
}()

var tesseractAvailable = func() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}()

func extractPDFPoppler(data []byte) (string, PDFStats, error) {
	st := PDFStats{Engine: "poppler"}
	dir, err := os.MkdirTemp("", "pdf-*")
	if err != nil {
		return "", st, err
	}
	defer os.RemoveAll(dir)
	src := filepath.Join(dir, "in.pdf")
	if err := os.WriteFile(src, data, 0o600); err != nil {
		return "", st, err
	}

	pages, err := pdfPageCount(src)
	if err != nil {
		return "", st, fmt.Errorf("pdfinfo: %w", err)
	}
	st.Pages = pages

	var b strings.Builder
	for p := 1; p <= pages; p++ {
		text, _ := runCmd("pdftotext", "-enc", "UTF-8", "-f", itoa(p), "-l", itoa(p), src, "-")
		text = cleanText(text)
		// A page is OCR'd when it has no usable text layer, or only a thin
		// one (e.g. a typed header over a scanned body): the longer result wins.
		if tesseractAvailable && (!usableText(text) || letterCount(text) < ocrIfBelow) {
			if ocr, oerr := ocrPage(dir, src, p); oerr == nil && usableText(ocr) &&
				letterCount(ocr) > letterCount(text)*13/10 {
				text = fixOCRNumbering(ocr)
				st.OCRPages++
			} else if oerr != nil {
				log.Printf("parse: OCR page %d: %v", p, oerr)
			}
		}
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	if st.OCRPages > 0 {
		st.Engine = "poppler+ocr"
	}
	out := normalizeExtracted(b.String())
	if strings.TrimSpace(out) == "" {
		return "", st, ErrNoText
	}
	return out, st, nil
}

func ocrPage(dir, src string, page int) (string, error) {
	prefix := filepath.Join(dir, "p"+itoa(page))
	if _, err := runCmd("pdftoppm", "-r", ocrDPI, "-gray", "-png", "-singlefile",
		"-f", itoa(page), "-l", itoa(page), src, prefix); err != nil {
		return "", fmt.Errorf("pdftoppm: %w", err)
	}
	img := prefix + ".png"
	defer os.Remove(img)
	// psm 4: single column of text of variable sizes — best for regulations.
	out, err := runCmd("tesseract", img, "-", "-l", "rus+eng", "--psm", "4")
	if err != nil {
		return "", fmt.Errorf("tesseract: %w", err)
	}
	return cleanText(out), nil
}

// usableText reports whether a page has a real text layer: enough letters
// and a sane share of letters among non-space characters.
func letterCount(s string) int {
	n := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			n++
		}
	}
	return n
}

func usableText(s string) bool {
	letters, nonSpace := 0, 0
	for _, r := range s {
		if unicode.IsSpace(r) {
			continue
		}
		nonSpace++
		if unicode.Is(unicode.Cyrillic, r) || unicode.Is(unicode.Latin, r) {
			letters++
		}
	}
	if letters < minPageChars {
		return false
	}
	return float64(letters)/float64(nonSpace) > 0.55
}

var (
	// OCR often drops the dot in "1.1 Текст" → "11 Текст". Restore it when a
	// line starts with two digits followed by a capitalised word.
	ocrNumRe   = regexp.MustCompile(`(?m)^(\d)(\d)\s+([А-ЯЁA-Z])`)
	hyphenWrap = regexp.MustCompile(`([а-яё])-\n([а-яё])`)
	manyBlank  = regexp.MustCompile(`\n{3,}`)
	spaces     = regexp.MustCompile(`[ \t\x{00A0}]+`)
)

func fixOCRNumbering(s string) string {
	return ocrNumRe.ReplaceAllString(s, "$1.$2. $3")
}

// normalizeExtracted joins hyphenated line wraps, collapses whitespace and
// drops page-number-only lines.
func normalizeExtracted(s string) string {
	s = strings.ReplaceAll(s, "\f", "\n")
	s = hyphenWrap.ReplaceAllString(s, "$1$2")
	lines := strings.Split(s, "\n")
	out := lines[:0]
	for _, l := range lines {
		l = strings.TrimSpace(spaces.ReplaceAllString(l, " "))
		if _, err := strconv.Atoi(l); err == nil && len(l) <= 3 {
			continue // page number
		}
		out = append(out, l)
	}
	return strings.TrimSpace(manyBlank.ReplaceAllString(strings.Join(out, "\n"), "\n\n"))
}

func pdfPageCount(src string) (int, error) {
	out, err := runCmd("pdfinfo", src)
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "Pages:") {
			return strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Pages:")))
		}
	}
	return 0, fmt.Errorf("page count not found")
}

func runCmd(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), pageTimeout)
	defer cancel()
	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

func itoa(i int) string { return strconv.Itoa(i) }
