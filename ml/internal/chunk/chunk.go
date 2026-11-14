// Package chunk splits regulation documents into searchable fragments.
// It prefers structural boundaries (numbered пункты, разделы, заголовки)
// so that each chunk maps to a citable section of the source document.
// Lines like "=== Страница N ===" (emitted by the OCR/PDF pipeline) set the
// page number carried by subsequent chunks.
package chunk

import (
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"phystech-answer/ml/internal/model"
)

const (
	// maxChunkRunes is the target upper bound for a single chunk.
	maxChunkRunes = 1400
	// overlapRunes keeps context continuity when a long section is split.
	overlapRunes = 200
	// minChunkRunes: pieces smaller than this are merged into a neighbour.
	minChunkRunes = 250
)

var (
	// "3.4.", "3.4 ", "1.2.3" — numeric prefix of a пункт.
	numPrefix = regexp.MustCompile(`^\s*(\d+(\.\d+)*\.?)`)
	// "3.4. Текст", "3.4 Текст" — numbered пункты of Russian regulations.
	numHeading = regexp.MustCompile(`^\s*\d+(\.\d+)*\.?[\s\x{00A0}]+\S`)
	// "Раздел 2", "Глава 1", "Статья 5", "Приложение № 1".
	wordHeading = regexp.MustCompile(`(?i)^\s*(раздел|глава|статья|приложение|часть|порядок|регламент|приказ)\b`)
	// Markdown headings.
	mdHeading = regexp.MustCompile(`^\s*#{1,6}\s`)
	// "=== Страница 12 ===" — page marker produced by parse_corpus.sh.
	pageMarker = regexp.MustCompile(`^\s*={2,}\s*Страница\s+(\d+)\s*={2,}\s*$`)
	// "--- page 5 ---" style markers sometimes found in extracted text.
	pageMarker2 = regexp.MustCompile(`^\s*-+\s*page\s+(\d+)\s*-+\s*$`)
)

// Split breaks doc into chunks. Every chunk carries the section heading it
// belongs to (e.g. "3.4") and, when page markers are present, the page
// number for citations.
func Split(doc model.Document) []model.Chunk {
	// Strip page markers but remember page boundaries per line.
	var lines []pline
	page := 0
	for _, raw := range strings.Split(normalize(doc.Content), "\n") {
		if m := pageMarker.FindStringSubmatch(raw); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				page = n
			}
			continue
		}
		if m := pageMarker2.FindStringSubmatch(raw); m != nil {
			if n, err := strconv.Atoi(m[1]); err == nil {
				page = n
			}
			continue
		}
		lines = append(lines, pline{text: raw, page: page})
	}

	// Reassemble text with page tracking: emit each section body as a
	// sequence of (page,text) runs so oversized sections keep page info.
	var out []model.Chunk
	for _, sec := range splitSectionsWithPages(lines) {
		for _, piece := range splitPiecesPageAware(sec, maxChunkRunes, overlapRunes) {
			text := strings.TrimSpace(piece.text)
			if text == "" {
				continue
			}
			out = append(out, model.Chunk{
				DocID:    doc.ID,
				DocTitle: doc.Title,
				Tags:     doc.Tags,
				Section:  sec.title,
				Page:     piece.page,
				Text:     text,
			})
		}
	}
	if len(out) == 0 {
		return nil
	}

	// Merge tiny pieces into the previous chunk of the same section, but
	// never grow a chunk past the size bound.
	merged := out[:0]
	for _, c := range out {
		last := len(merged) - 1
		if last >= 0 &&
			utf8.RuneCountInString(c.Text) < minChunkRunes &&
			merged[last].Section == c.Section &&
			merged[last].Page == c.Page &&
			utf8.RuneCountInString(merged[last].Text)+utf8.RuneCountInString(c.Text)+1 <= maxChunkRunes {
			merged[last].Text += "\n" + c.Text
			continue
		}
		merged = append(merged, c)
	}
	return merged
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// isHeading reports whether a line looks like a structural heading of a
// regulation: a numbered пункт, a section word, markdown heading, or a
// fully upper-case title line.
func isHeading(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	return mdHeading.MatchString(trimmed) ||
		wordHeading.MatchString(trimmed) ||
		numHeading.MatchString(trimmed) ||
		isAllCapsTitle(trimmed)
}

func isAllCapsTitle(s string) bool {
	if n := utf8.RuneCountInString(s); n < 8 || n > 120 {
		return false
	}
	var letters int
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsLower(r) {
				return false
			}
		}
	}
	return letters >= 5
}

// headingTitle extracts a short citation label for a heading line. Long
// numbered пункты collapse to their numeric prefix ("3.4." lir… "3.4.").
func headingTitle(line string) string {
	t := strings.TrimSpace(line)
	t = strings.TrimLeft(t, "#")
	t = strings.TrimSpace(t)
	if utf8.RuneCountInString(t) > 60 {
		if m := numPrefix.FindStringSubmatch(t); m != nil && numHeading.MatchString(t) {
			return strings.TrimSpace(m[0])
		}
		if r := []rune(t); len(r) > 60 {
			return strings.TrimSpace(string(r[:57])) + "…"
		}
	}
	return t
}

type pline struct {
	text string
	page int
}

type psection struct {
	title string
	lines []pline
}

// splitSectionsWithPages groups lines into sections led by headings,
// keeping the page number of each line for citation mapping.
func splitSectionsWithPages(lines []pline) []psection {
	var sections []psection
	var cur *psection
	flush := func() {
		if cur == nil {
			return
		}
		if len(cur.lines) > 0 {
			sections = append(sections, *cur)
		}
		cur = nil
	}

	for _, l := range lines {
		if isHeading(l.text) {
			flush()
			cur = &psection{title: headingTitle(l.text)}
		}
		if cur == nil {
			cur = &psection{}
		}
		cur.lines = append(cur.lines, l)
	}
	flush()
	return sections
}

type pendingPiece struct {
	text string
	page int
}

// splitPiecesPageAware splits a section into pieces of at most maxRunes,
// preferring paragraph/sentence/word boundaries. The page of a piece is the
// page of its first line.
func splitPiecesPageAware(sec psection, maxRunes, overlap int) []pendingPiece {
	// Rebuild paragraphs with page tracking.
	type para struct {
		text string
		page int
	}
	var paras []para
	var buf strings.Builder
	bufPage := 0
	flushPara := func() {
		if strings.TrimSpace(buf.String()) != "" {
			paras = append(paras, para{text: buf.String(), page: bufPage})
		}
		buf.Reset()
	}
	for _, l := range sec.lines {
		if buf.Len() == 0 {
			bufPage = l.page
		}
		if strings.TrimSpace(l.text) == "" {
			flushPara()
			continue
		}
		if buf.Len() > 0 {
			buf.WriteString("\n")
		}
		buf.WriteString(l.text)
	}
	flushPara()

	var out []pendingPiece
	var cur strings.Builder
	curLen, curPage := 0, 0
	flush := func() {
		if strings.TrimSpace(cur.String()) != "" {
			out = append(out, pendingPiece{text: cur.String(), page: curPage})
		}
		cur.Reset()
		curLen = 0
	}
	for _, p := range paras {
		text := strings.TrimSpace(p.text)
		plen := utf8.RuneCountInString(text)
		if plen == 0 {
			continue
		}
		if plen > maxRunes {
			flush()
			for _, sub := range splitLongP(text, maxRunes, overlap) {
				out = append(out, pendingPiece{text: sub, page: p.page})
			}
			continue
		}
		if curLen > 0 && curLen+2+plen > maxRunes {
			flush()
		}
		if curLen == 0 {
			curPage = p.page
		}
		if curLen > 0 {
			cur.WriteString("\n\n")
			curLen += 2
		}
		cur.WriteString(text)
		curLen += plen
	}
	flush()
	return out
}

// splitLongP hard-splits a long text at boundary-preferred positions.
func splitLongP(s string, maxRunes, overlap int) []string {
	r := []rune(s)
	var out []string
	start := 0
	for len(r)-start > maxRunes {
		limit := start + maxRunes
		cut := limit
		lo := start + maxRunes*3/4
		for i := limit - 1; i >= lo; i-- {
			if r[i] == '\n' {
				cut = i
				break
			}
		}
		if cut == limit {
			for i := limit - 1; i >= lo; i-- {
				if r[i] == ' ' || r[i] == '.' || r[i] == '!' || r[i] == '?' {
					cut = i + 1
					break
				}
			}
		}
		piece := strings.TrimSpace(string(r[start:cut]))
		if piece != "" {
			out = append(out, piece)
		}
		next := cut - overlap
		if next < start+minChunkRunes/2 {
			next = start + minChunkRunes/2
		}
		for next < cut && next < len(r) && r[next] != ' ' && r[next] != '\n' {
			next++
		}
		start = next
	}
	if tail := strings.TrimSpace(string(r[start:])); tail != "" {
		out = append(out, tail)
	}
	return out
}
