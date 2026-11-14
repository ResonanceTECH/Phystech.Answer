// Package relations extracts normative links between documents from their
// text: repeals, amendments, approvals, references, attachments.
//
// Two tiers:
//  1. Deterministic regex pass — fast, high precision for standard phrasings
//     («Считать утратившим силу приказ … № N от DD.MM.YYYY»).
//  2. Optional LLM pass — for ambiguous references (used by the API endpoint,
//     not at index time).
package relations

import (
	"regexp"
	"strings"

	"phystech-answer/ml/internal/model"
)

var (
	repealRe = regexp.MustCompile(`(?i)считать\s+утративш\w*\s+силу[^.\n]{0,200}?приказ\w*[^№]{0,40}№\s*([\d\-/]+)\s+от\s+(\d{2})\.(\d{2})\.(\d{4})`)
	// «приказ ректора МФТИ № 76-1 от 12.02.2015» without "считать утратившим"
	amendRe      = regexp.MustCompile(`(?i)(?:изложить|изменить|дополнить)[^.\n]{0,120}?(?:приказ\w*|пункт\w*)[^№]{0,40}№\s*([\d\-/]+)\s+от\s+(\d{2})\.(\d{2})\.(\d{4})`)
	approveRe    = regexp.MustCompile(`(?i)утвердить[^.\n]{0,120}?(?:приложени\w*|регламент\w*|порядок\w*)`)
	genericRefRe = regexp.MustCompile(`(?i)(?:в\s+соответствии\s+с|на\s+основании|в\s+порядке)[^.\n]{0,80}?(?:приказ\w*|распоряжени\w*)[^№]{0,40}№\s*([\d\-/]+)\s+от\s+(\d{2})\.(\d{2})\.(\d{4})`)
	docNumRe     = regexp.MustCompile(`(?i)(?:приказ\w*|распоряжени\w*)\s+(?:ректора\s+МФТИ\s+)?№\s*([\d\-/]+)`)
)

// Extract scans doc content for relation patterns. evidence holds the raw
// sentence for auditability.
func Extract(doc model.Document) []model.Relation {
	var out []model.Relation
	seen := map[string]bool{}

	add := func(relType string, num, date, evidence string) {
		num = strings.TrimSpace(num)
		key := relType + "|" + num + "|" + date
		if seen[key] {
			return
		}
		seen[key] = true
		rel := model.Relation{
			RelationType:         relType,
			TargetDocumentNumber: num,
			TargetDocumentDate:   date,
			Evidence:             strings.TrimSpace(evidence),
		}
		// Try to resolve the target inside the current corpus — the caller
		// fills TargetDocumentID when it matches a known document number.
		out = append(out, rel)
	}

	for _, para := range paragraphs(doc.Content) {
		for _, m := range repealRe.FindAllStringSubmatch(para, -1) {
			add("repeals", m[1], normDate(m[2], m[3], m[4]), para)
		}
		for _, m := range amendRe.FindAllStringSubmatch(para, -1) {
			add("amends", m[1], normDate(m[2], m[3], m[4]), para)
		}
		for _, m := range genericRefRe.FindAllStringSubmatch(para, -1) {
			add("references", m[1], normDate(m[2], m[3], m[4]), para)
		}
		if approveRe.MatchString(para) && strings.Contains(strings.ToLower(para), "утвердить") {
			add("approves", "", "", para)
		}
	}
	return out
}

func paragraphs(content string) []string {
	var out []string
	for _, p := range strings.Split(content, "\n") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func normDate(d, m, y string) string {
	return y + "-" + m + "-" + d
}

// ResolveTargets maps relation targets to document IDs by matching the
// target number/date against the corpus metadata.
func ResolveTargets(rels []model.Relation, docs []model.Document) []model.Relation {
	byNum := map[string]model.Document{}
	for _, d := range docs {
		if d.DocumentNumber != "" {
			byNum[normalizeNum(d.DocumentNumber)] = d
		}
	}
	for i, r := range rels {
		if d, ok := byNum[normalizeNum(r.TargetDocumentNumber)]; ok {
			rels[i].TargetDocumentID = d.ID
			rels[i].Confirmed = true
		}
	}
	return rels
}

func normalizeNum(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "№", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}

// DocNumberOf extracts the document's own number from metadata or title.
func DocNumberOf(d model.Document) string {
	if d.DocumentNumber != "" {
		return normalizeNum(d.DocumentNumber)
	}
	if m := docNumRe.FindStringSubmatch(d.Title); m != nil {
		return normalizeNum(m[1])
	}
	return ""
}
