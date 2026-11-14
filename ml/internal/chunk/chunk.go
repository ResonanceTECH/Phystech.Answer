// Package chunk splits regulation documents into searchable, citable
// fragments.
//
// Strategy (structure-aware chunking):
//  1. the text is cut into blocks at numbered пункты ("3.4.", "3.4 ") and
//     section headings ("3. Порядок оформления", "РЕГЛАМЕНТ …",
//     "Приложение № 1");
//  2. consecutive blocks of one section are packed into a chunk of
//     ~targetRunes, so a chunk is a coherent piece of a procedure rather
//     than a single sentence;
//  3. every chunk carries a citation label ("п. 3.1–3.4"), the section
//     heading and a context header (document, number, date, section) that
//     is prepended for BM25 and embeddings ("contextual chunk headers").
package chunk

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"phystech-answer/ml/internal/model"
)

const (
	targetRunes   = 900  // pack blocks until a chunk reaches this size
	maxChunkRunes = 1400 // hard upper bound for a chunk
	overlapRunes  = 160  // overlap when a single oversized block is split
	minChunkRunes = 200
)

var (
	// "3.4. Текст", "3.4 Текст", "3.4.1. Текст" — multi-level пункт.
	subPunkt = regexp.MustCompile(`^(\d{1,2}(?:\.\d{1,2}){1,3})\.?(?:[\s\x{00A0}]+(\S.*)?|([А-ЯЁA-Z«"(].*))$`)
	// "3. Текст" / "3) Текст" — first-level пункт or section heading.
	topPunkt = regexp.MustCompile(`^(\d{1,2})[.)][\s\x{00A0}]+(\S.*)$`)
	// "Раздел 2", "Глава 1", "Приложение № 1", "Статья 5".
	wordHeading = regexp.MustCompile(`(?i)^(раздел|глава|статья|приложение|часть)\b`)
	// Lone number line produced by PDF column layout ("1.1." on its own).
	loneNumber = regexp.MustCompile(`^(\d{1,2}(?:\.\d{1,2}){0,3})\.?$`)
)

type block struct {
	number  string // "3.4", "" for unnumbered
	heading string // section heading this block belongs to
	text    strings.Builder
}

// Split breaks doc into chunks.
func Split(doc model.Document) []model.Chunk {
	text := normalize(doc.Content)
	if strings.TrimSpace(text) == "" {
		return nil
	}
	blocks := splitBlocks(text)

	var out []model.Chunk
	var cur []*block
	curLen := 0
	emit := func() {
		if len(cur) == 0 {
			return
		}
		var b strings.Builder
		for i, bl := range cur {
			if i > 0 {
				b.WriteString("\n")
			}
			b.WriteString(strings.TrimSpace(bl.text.String()))
		}
		body := strings.TrimSpace(b.String())
		label := label(cur)
		heading := cur[0].heading
		for _, piece := range splitPieces(body, maxChunkRunes, overlapRunes) {
			if piece = strings.TrimSpace(piece); piece == "" {
				continue
			}
			out = append(out, model.Chunk{
				DocID:     doc.ID,
				DocTitle:  doc.Title,
				DocNumber: doc.Number,
				DocDate:   doc.Date,
				DocType:   doc.Type,
				Tags:      doc.Tags,
				Section:   label,
				Heading:   heading,
				Text:      piece,
			})
		}
		cur, curLen = nil, 0
	}

	for _, bl := range blocks {
		n := utf8.RuneCountInString(bl.text.String())
		sameSection := len(cur) > 0 && cur[0].heading == bl.heading
		if len(cur) > 0 && (!sameSection || curLen+n > maxChunkRunes || curLen >= targetRunes) {
			// Keep tiny tails attached instead of emitting a 1-line chunk.
			if !(sameSection && curLen < minChunkRunes && curLen+n <= maxChunkRunes) {
				emit()
			}
		}
		cur = append(cur, bl)
		curLen += n
	}
	emit()

	for i := range out {
		out[i].Context = contextHeader(out[i])
	}
	return out
}

// splitBlocks cuts the text into numbered blocks and tracks headings.
func splitBlocks(text string) []*block {
	var blocks []*block
	heading := ""
	var cur *block
	pendingNumber := ""

	start := func(number string) {
		cur = &block{number: number, heading: heading}
		blocks = append(blocks, cur)
	}

	var lines []string
	for _, raw := range strings.Split(text, "\n") {
		if l := strings.TrimSpace(raw); l != "" {
			lines = append(lines, l)
		}
	}
	for i, line := range lines {
		next := ""
		if i+1 < len(lines) {
			next = lines[i+1]
		}
		// PDF layouts sometimes put "1.1." on its own line before the text.
		if m := loneNumber.FindStringSubmatch(line); m != nil {
			pendingNumber = m[1]
			continue
		}
		if pendingNumber != "" {
			line = pendingNumber + ". " + line
			pendingNumber = ""
		}

		switch {
		case isSectionHeading(line) && !continues(next):
			heading = headingTitle(line)
			start(numberOf(line))
		case subPunkt.MatchString(line):
			start(subPunkt.FindStringSubmatch(line)[1])
		case topPunkt.MatchString(line):
			start(topPunkt.FindStringSubmatch(line)[1])
		case cur == nil:
			start("")
		}
		cur.text.WriteString(line)
		cur.text.WriteString("\n")
	}
	return blocks
}

// continues reports whether the next line carries on the same sentence
// (PDF line wrap): it starts with a lower-case letter.
func continues(next string) bool {
	r, _ := utf8.DecodeRuneInString(next)
	return unicode.IsLower(r)
}

// isSectionHeading: "3. Порядок оформления командировки" (short, no final
// punctuation), "РЕГЛАМЕНТ …" in caps, or "Приложение № 1".
func isSectionHeading(line string) bool {
	n := utf8.RuneCountInString(line)
	if n > 120 {
		return false
	}
	if wordHeading.MatchString(line) && n < 80 {
		return true
	}
	if m := topPunkt.FindStringSubmatch(line); m != nil {
		rest := strings.TrimSpace(m[2])
		first, _ := utf8.DecodeRuneInString(rest)
		if !unicode.IsUpper(first) || utf8.RuneCountInString(rest) > 90 {
			return false
		}
		// "Порядок бронирования Концертного зала." — a title may end with a
		// dot, but must be a single short phrase, not a sentence.
		body := strings.TrimRight(rest, ".")
		last, _ := utf8.DecodeLastRuneInString(body)
		words := len(strings.Fields(body))
		if strings.HasSuffix(rest, ".") && (words > 6 || strings.ContainsAny(body, "0123456789")) {
			return false // a short sentence ("Срок … от 1 до 4 месяцев."), not a title
		}
		return !strings.ContainsRune(";:,", last) && !strings.ContainsAny(body, ".;:") && words <= 10
	}
	return isAllCapsTitle(line)
}

func isAllCapsTitle(s string) bool {
	if n := utf8.RuneCountInString(s); n < 8 || n > 120 {
		return false
	}
	letters := 0
	for _, r := range s {
		if unicode.IsLetter(r) {
			letters++
			if unicode.IsLower(r) {
				return false
			}
		}
	}
	return letters >= 6
}

func numberOf(line string) string {
	if m := topPunkt.FindStringSubmatch(line); m != nil {
		return m[1]
	}
	return ""
}

func headingTitle(line string) string {
	t := strings.TrimSpace(strings.TrimLeft(line, "#"))
	if r := []rune(t); len(r) > 90 {
		return strings.TrimSpace(string(r[:87])) + "…"
	}
	return t
}

// label builds the citation label for a group of blocks: "п. 3.2" or
// "п. 3.1–3.4"; falls back to the section heading.
func label(bs []*block) string {
	first, last := "", ""
	for _, b := range bs {
		if b.number == "" {
			continue
		}
		if first == "" {
			first = b.number
		}
		// Keep the range monotonic: numbering restarts inside a chunk (e.g.
		// list items "1 - 5 разряд" after "2.2") would produce "п. 2–11".
		if last == "" || (numLess(last, b.number) && topLevel(b.number)-topLevel(last) <= 1) {
			last = b.number
		}
	}
	switch {
	case first == "":
		return bs[0].heading
	case first == last:
		return "п. " + first
	default:
		return "п. " + first + "–" + last
	}
}

func numParts(n string) []int {
	var out []int
	for _, p := range strings.Split(n, ".") {
		v := 0
		for _, r := range p {
			v = v*10 + int(r-'0')
		}
		out = append(out, v)
	}
	return out
}

func numLess(a, b string) bool {
	if a == "" {
		return true
	}
	x, y := numParts(a), numParts(b)
	for i := 0; i < len(x) && i < len(y); i++ {
		if x[i] != y[i] {
			return x[i] < y[i]
		}
	}
	return len(x) < len(y)
}

func topLevel(n string) int { return numParts(n)[0] }

func contextHeader(c model.Chunk) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Документ: %s", c.DocTitle)
	if c.DocNumber != "" {
		fmt.Fprintf(&b, " № %s", c.DocNumber)
	}
	if c.DocDate != "" {
		fmt.Fprintf(&b, " от %s", c.DocDate)
	}
	if c.Heading != "" && c.Heading != c.DocTitle {
		fmt.Fprintf(&b, ". Раздел: %s", c.Heading)
	}
	if c.Section != "" && c.Section != c.Heading {
		fmt.Fprintf(&b, ". %s", c.Section)
	}
	return b.String()
}

func normalize(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	return strings.TrimSpace(s)
}

// splitPieces splits s into chunks of at most maxRunes, preferring breaks at
// line, sentence or word boundaries, with a small overlap.
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
	for _, p := range strings.Split(s, "\n") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		plen := utf8.RuneCountInString(p)
		if plen > maxRunes {
			flush()
			pieces = append(pieces, splitLong(p, maxRunes, overlap)...)
			continue
		}
		if curLen > 0 && curLen+1+plen > maxRunes {
			flush()
		}
		if curLen > 0 {
			cur.WriteString("\n")
			curLen++
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
		lo := start + maxRunes*3/4
		for i := limit - 1; i >= lo; i-- {
			if r[i] == '.' || r[i] == ';' || r[i] == '!' || r[i] == '?' {
				cut = i + 1
				break
			}
		}
		if cut == limit {
			for i := limit - 1; i >= lo; i-- {
				if r[i] == ' ' {
					cut = i + 1
					break
				}
			}
		}
		if piece := strings.TrimSpace(string(r[start:cut])); piece != "" {
			out = append(out, piece)
		}
		next := cut - overlap
		if next < start+minChunkRunes/2 {
			next = start + minChunkRunes/2
		}
		for next < cut && next < len(r) && r[next] != ' ' {
			next++
		}
		start = next
	}
	if tail := strings.TrimSpace(string(r[start:])); tail != "" {
		out = append(out, tail)
	}
	return out
}
