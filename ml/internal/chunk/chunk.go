// Package chunk splits regulation documents into searchable fragments.
// It prefers structural boundaries (numbered пункты, разделы, заголовки)
// so that each chunk maps to a citable section of the source document.
package chunk

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"phystech-answer/ml/internal/model"
)

const (
	// maxChunkRunes is the target upper bound for a single chunk.
	maxChunkRunes = 1200
	// overlapRunes keeps context continuity when a long section is split.
	overlapRunes = 160
	// minChunkRunes: pieces smaller than this are merged into a neighbour.
	minChunkRunes = 200
)

var (
	// "3.4.", "3.4 ", "1.2.3" — numeric prefix of a пункт.
	numPrefix = regexp.MustCompile(`^\s*(\d+(\.\d+)*\.?)`)
	// "3.4. Текст", "3.4 Текст" — numbered пункты of Russian regulations.
	numHeading = regexp.MustCompile(`^\s*\d+(\.\d+)*\.?[\s\x{00A0}]+\S`)
	// "Раздел 2", "Глава 1", "Статья 5", "Приложение № 1".
	wordHeading = regexp.MustCompile(`(?i)^\s*(раздел|глава|статья|приложение|часть)\b`)
	// Markdown headings.
	mdHeading = regexp.MustCompile(`^\s*#{1,6}\s`)
)

type section struct {
	title string
	body  strings.Builder
}

// Split breaks doc into chunks. Every chunk carries the section heading it
// belongs to (e.g. "3.4") for citations.
func Split(doc model.Document) []model.Chunk {
	text := normalize(doc.Content)
	if strings.TrimSpace(text) == "" {
		return nil
	}

	var out []model.Chunk
	for _, sec := range splitSections(text) {
		for _, piece := range splitPieces(sec.body.String(), maxChunkRunes, overlapRunes) {
			piece = strings.TrimSpace(piece)
			if piece == "" {
				continue
			}
			out = append(out, model.Chunk{
				DocID:    doc.ID,
				DocTitle: doc.Title,
				Tags:     doc.Tags,
				Section:  sec.title,
				Text:     piece,
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

// splitSections groups lines into sections led by headings. Heading lines
// are kept inside the section body as well — they carry meaningful terms
// ("Общие положения", "Командировки") and preserving them avoids losing
// short пункты that look like headings. A document without headings yields
// a single anonymous section.
func splitSections(text string) []section {
	var sections []section
	var cur *section
	var buf strings.Builder

	flush := func() {
		if cur == nil {
			return
		}
		if b := strings.TrimSpace(buf.String()); b != "" {
			cur.body.WriteString(b)
			cur.body.WriteString("\n")
		}
		buf.Reset()
	}

	for _, line := range strings.Split(text, "\n") {
		if isHeading(line) {
			flush()
			if cur != nil {
				sections = append(sections, *cur)
			}
			cur = &section{title: headingTitle(line)}
		}
		if cur == nil {
			cur = &section{}
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	flush()
	if cur != nil {
		sections = append(sections, *cur)
	}
	return sections
}

// splitPieces splits s into chunks of at most maxRunes, preferring breaks at
// paragraph, sentence or word boundaries, with a small overlap.
func splitPieces(s string, maxRunes, overlap int) []string {
	if utf8.RuneCountInString(strings.TrimSpace(s)) <= maxRunes {
		return []string{s}
	}

	var pieces []string
	var cur strings.Builder
	curLen := 0

	flush := func() {
		if strings.TrimSpace(cur.String()) != "" {
			pieces = append(pieces, cur.String())
		}
		cur.Reset()
		curLen = 0
	}

	for _, p := range strings.Split(s, "\n\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		plen := utf8.RuneCountInString(p)
		if plen > maxRunes {
			// Oversized paragraph: flush current, hard-split the paragraph.
			flush()
			pieces = append(pieces, splitLong(p, maxRunes, overlap)...)
			continue
		}
		if curLen > 0 && curLen+2+plen > maxRunes {
			flush()
		}
		if curLen > 0 {
			cur.WriteString("\n\n")
			curLen += 2
		}
		cur.WriteString(p)
		curLen += plen
	}
	flush()
	return pieces
}

// splitLong hard-splits a long text at boundary-preferred positions.
func splitLong(s string, maxRunes, overlap int) []string {
	r := []rune(s)
	var out []string
	start := 0
	for len(r)-start > maxRunes {
		limit := start + maxRunes
		cut := limit
		// Prefer the last newline in the last quarter of the window.
		lo := start + maxRunes*3/4
		for i := limit - 1; i >= lo; i-- {
			if r[i] == '\n' {
				cut = i
				break
			}
		}
		if cut == limit {
			// Then a sentence or word boundary.
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
		// Next window starts `overlap` runes back, snapped forward to a space.
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
