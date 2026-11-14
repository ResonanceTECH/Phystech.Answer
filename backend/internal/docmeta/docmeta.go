// Package docmeta extracts normative metadata from regulation filenames and
// text: document type (приказ/регламент/распоряжение/приложение), number,
// date and references to documents it amends. The metadata drives version
// awareness in retrieval ("which act is newer / which one changes which").
package docmeta

import (
	"regexp"
	"strings"
	"time"
)

type Meta struct {
	Title  string
	Type   string     // приказ|распоряжение|регламент|приложение|форма|положение
	Number string     // "794-1", "700"
	Date   *time.Time // дата документа
	// Amends lists numbers of acts this document changes ("о внесении
	// изменений в приказ … № 3065-1").
	Amends []string
}

var (
	numRe  = regexp.MustCompile(`(?i)(?:№|N|No\.?)\s*(\d+(?:[-/]\d+)?(?:-[а-яa-z]+)?)`)
	dateRe = regexp.MustCompile(`(?:от\s+)?(\d{1,2})\.(\d{1,2})\.(\d{4})`)
	// "от 3 ноября 2026 г."
	dateWordsRe = regexp.MustCompile(`(?i)от\s+[«"]?(\d{1,2})[»"]?\s+(января|февраля|марта|апреля|мая|июня|июля|августа|сентября|октября|ноября|декабря)\s+(\d{4})`)
	amendRe     = regexp.MustCompile(`(?is)(?:изменени[йяе]|дополнени[йя]|отмен[аеиу]|утратившим[и]? силу)[^.]{0,200}?(?:№|N)\s*(\d+(?:-\d+)?)`)
	// Filenames prefixed with a bare number: "460-1 Открытие ФЛС_…".
	leadingNumRe = regexp.MustCompile(`^(\d+-\d+)\s`)
)

var months = map[string]time.Month{
	"января": 1, "февраля": 2, "марта": 3, "апреля": 4, "мая": 5, "июня": 6,
	"июля": 7, "августа": 8, "сентября": 9, "октября": 10, "ноября": 11, "декабря": 12,
}

// FromFilename derives a human title and metadata from a file name such as
// "Приказ N 794-1 от 28.03.2023.docx".
func FromFilename(filename string) Meta {
	base := filename
	if i := strings.LastIndexByte(base, '.'); i > 0 {
		base = base[:i]
	}
	base = strings.TrimSpace(strings.TrimPrefix(base, "Копия "))
	base = strings.TrimRight(base, ". ")
	base = strings.Join(strings.Fields(strings.ReplaceAll(base, "_", " ")), " ")

	m := Meta{Title: base, Type: detectType(base)}
	if mm := numRe.FindStringSubmatch(base); mm != nil {
		m.Number = mm[1]
	} else if mm := leadingNumRe.FindStringSubmatch(base); mm != nil {
		m.Number = mm[1]
	}
	m.Date = findDate(base)
	return m
}

// Enrich fills missing fields from the document text (header area) and
// collects amendment references from the whole text.
func Enrich(m Meta, text string) Meta {
	head := text
	if r := []rune(head); len(r) > 2500 {
		head = string(r[:2500])
	}
	if m.Type == "" {
		m.Type = detectType(head)
	}
	// Only trust "от <дата> № <номер>" / "№ <номер> от <дата>" pairs: bare
	// numbers and dates in the header often belong to referenced laws.
	if num, date := findRequisites(head, m.Number); num != "" || date != nil {
		if m.Number == "" {
			m.Number = num
		}
		if m.Date == nil {
			m.Date = date
		}
	}
	seen := map[string]bool{m.Number: true}
	for _, mm := range amendRe.FindAllStringSubmatch(text, -1) {
		if n := mm[1]; !seen[n] {
			seen[n] = true
			m.Amends = append(m.Amends, n)
		}
	}
	return m
}

var (
	reqDate = `(\d{1,2}\.\d{1,2}\.\d{4})`
	reqNum  = `(\d+(?:-\d+)?)`
	// "от 19.06.2026 № 2993-1" (possibly across line breaks)
	reqDateNumRe = regexp.MustCompile(`(?s)от\s*` + reqDate + `\s*(?:г\.?)?\s{0,20}(?:№|N)\s*` + reqNum)
	// "№ 794-1 от 28.03.2023"
	reqNumDateRe = regexp.MustCompile(`(?:№|N)\s*` + reqNum + `\s*от\s*` + reqDate)
	refContext   = regexp.MustCompile(`(?i)(утратившим|утратившими|изменени|в редакции|пункт|приказа ректора|приказом ректора)`)
	lawContext   = regexp.MustCompile(`(?i)(закон|ФЗ|кодекс|постановлени|министерств|РФ\b|Российской Федерации)`)
)

// findRequisites returns the first number/date pair that is not part of a
// reference to an external law.
func findRequisites(head, knownNumber string) (string, *time.Time) {
	type cand struct {
		start     int
		num, date string
	}
	var cs []cand
	for _, mm := range reqDateNumRe.FindAllStringSubmatchIndex(head, -1) {
		cs = append(cs, cand{mm[0], head[mm[4]:mm[5]], head[mm[2]:mm[3]]})
	}
	for _, mm := range reqNumDateRe.FindAllStringSubmatchIndex(head, -1) {
		cs = append(cs, cand{mm[0], head[mm[2]:mm[3]], head[mm[4]:mm[5]]})
	}
	best := -1
	for i, c := range cs {
		from := c.start - 240
		if from < 0 {
			from = 0
		}
		end := c.start + 120
		if end > len(head) {
			end = len(head)
		}
		if lawContext.MatchString(head[from:end]) {
			continue
		}
		if knownNumber != "" && c.num != knownNumber {
			continue // requisites of another (referenced) act
		}
		near := c.start - 160
		if near < 0 {
			near = 0
		}
		if refContext.MatchString(head[near:c.start]) {
			continue
		}
		if best < 0 || c.start < cs[best].start {
			best = i
		}
	}
	if best < 0 {
		return "", nil
	}
	return cs[best].num, findDate(cs[best].date)
}

func detectType(s string) string {
	l := strings.ToLower(s)
	switch {
	case strings.HasPrefix(l, "приложени"):
		return "приложение"
	case strings.Contains(l, "бланк") || strings.Contains(l, "смет") && strings.Contains(l, "форм"):
		return "форма"
	case strings.HasPrefix(l, "распоряжени"):
		return "распоряжение"
	case strings.HasPrefix(l, "приказ") || numRe.MatchString(l) && strings.Contains(l, "приказ"):
		return "приказ"
	case strings.Contains(l, "регламент"):
		return "регламент"
	case strings.Contains(l, "положени"):
		return "положение"
	case strings.Contains(l, "приказ"):
		return "приказ"
	}
	return ""
}

func findDate(s string) *time.Time {
	if mm := dateRe.FindStringSubmatch(s); mm != nil {
		if t, err := time.Parse("2.1.2006", mm[1]+"."+mm[2]+"."+mm[3]); err == nil {
			return &t
		}
	}
	if mm := dateWordsRe.FindStringSubmatch(s); mm != nil {
		mon := months[strings.ToLower(mm[2])]
		var d, y int
		for _, c := range mm[1] {
			d = d*10 + int(c-'0')
		}
		for _, c := range mm[3] {
			y = y*10 + int(c-'0')
		}
		t := time.Date(y, mon, d, 0, 0, 0, 0, time.UTC)
		return &t
	}
	return nil
}

// Folders are thematic groups of the knowledge base; an administrator can
// move a document to another folder later.
var folderRules = []struct {
	name string
	re   *regexp.Regexp
}{
	{"Формы и бланки", regexp.MustCompile(`(?i)(бланк|^копия приложение|^приложение 1)`)},
	{"Финансы проектов", regexp.MustCompile(`(?i)(флс|лицев|смет|заимств)`)},
	{"Бюджет и финансовая структура", regexp.MustCompile(`(?i)(бюджет|фин\.? ?структур|(?:^|[\s«])эб(?:$|[\s»]))`)},
	{"Кадры и командировки", regexp.MustCompile(`(?i)(подбор|персонал|командир|кадр)`)},
	{"ИТ и инфраструктура", regexp.MustCompile(`(?i)(техподдерж|технической поддерж|обработки данных|цод|информационн)`)},
	{"Кампус и мероприятия", regexp.MustCompile(`(?i)(поселени|общежит|зал|мероприят)`)},
}

// SuggestFolder picks a thematic folder from the document title.
func SuggestFolder(title string) string {
	for _, r := range folderRules {
		if r.re.MatchString(title) {
			return r.name
		}
	}
	return "Прочие документы"
}

// RefineFolder is applied after parsing: when the title gave no folder, the
// document header text decides (forms are recognised by title only).
func RefineFolder(current, title, text string) string {
	if current != "" && current != "Прочие документы" {
		return current
	}
	if f := SuggestFolder(title); f != "Прочие документы" {
		return f
	}
	head := text
	if r := []rune(head); len(r) > 1500 {
		head = string(r[:1500])
	}
	for _, r := range folderRules[1:] {
		if r.re.MatchString(head) {
			return r.name
		}
	}
	return "Прочие документы"
}
