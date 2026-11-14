package parse

import (
	"regexp"
	"strings"
)

// Section is a numbered fragment of a regulation, e.g. "3.4" with its body.
type Section struct {
	Number   string // "3", "3.4", "" for unnumbered preamble
	Title    string // first line, trimmed
	Content  string // full text of the fragment
	Position int
}

var numberedItem = regexp.MustCompile(`^\s*(\d+(?:\.\d+)*)\.\s`)

// SplitSections splits regulation text into numbered пункты. A line like
// "3.4. Текст" starts a new fragment; everything before the first numbered
// line and unnumbered paragraphs attach to the current fragment.
func SplitSections(text string) []Section {
	var out []Section
	var cur *Section
	flush := func() {
		if cur == nil {
			return
		}
		cur.Content = strings.TrimSpace(cur.Content)
		if cur.Content != "" {
			out = append(out, *cur)
		}
		cur = nil
	}

	for _, line := range strings.Split(text, "\n") {
		trimmed := strings.TrimSpace(line)
		if m := numberedItem.FindStringSubmatch(trimmed); m != nil {
			flush()
			cur = &Section{Number: m[1], Title: firstLineTitle(trimmed)}
		}
		if cur == nil {
			cur = &Section{}
		}
		cur.Content += trimmed + "\n"
	}
	flush()

	for i := range out {
		out[i].Position = i
		if out[i].Title == "" {
			out[i].Title = firstLineTitle(out[i].Content)
		}
	}
	return out
}

func firstLineTitle(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	line = strings.TrimSpace(line)
	if r := []rune(line); len(r) > 120 {
		return string(r[:117]) + "…"
	}
	return line
}
