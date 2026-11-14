package docmeta

import "testing"

func TestFromFilename(t *testing.T) {
	cases := []struct{ in, typ, num, date string }{
		{"Приказ N 794-1 от 28.03.2023.docx", "приказ", "794-1", "2023-03-28"},
		{"Распоряжение N 700 от 28.12.2024 Порядок присвоения номера ФЛС.docx", "распоряжение", "700", "2024-12-28"},
		{"460-1 Открытие ФЛС_приказ об изменении в приказе на открытие ФЛС.pdf", "приказ", "460-1", ""},
		{"Приказ №  5990-1 от 03.11.2026 новый.pdf", "приказ", "5990-1", "2026-11-03"},
		{"Регламент о командировании.pdf", "регламент", "", ""},
	}
	for _, c := range cases {
		m := FromFilename(c.in)
		d := ""
		if m.Date != nil {
			d = m.Date.Format("2006-01-02")
		}
		if m.Type != c.typ || m.Number != c.num || d != c.date {
			t.Errorf("%s: got %q %q %q", c.in, m.Type, m.Number, d)
		}
	}
}

func TestEnrichIgnoresLawReferences(t *testing.T) {
	text := `РЕГЛАМЕНТ поиска персонала разработан в соответствии с Законом РФ "О СМИ" от 27.12.1991 г. № 2124-1`
	m := Enrich(Meta{}, text)
	if m.Number != "" || m.Date != nil {
		t.Fatalf("law reference taken as requisites: %q %v", m.Number, m.Date)
	}
	m = Enrich(Meta{}, "Приказ\nот 19.06.2026\n\n№ 2993-1\n\nОб утверждении Регламента")
	if m.Number != "2993-1" || m.Date == nil || m.Date.Format("2006-01-02") != "2026-06-19" {
		t.Fatalf("requisites not found: %q %v", m.Number, m.Date)
	}
	m = Enrich(Meta{Number: "460-1"}, "ПРИКАЗ\n1. Считать утратившим силу приказ ректора МФТИ № 138-1 от 19.03.2015г.")
	if m.Date != nil {
		t.Fatalf("referenced act date taken: %v", m.Date)
	}
	m = Enrich(Meta{}, "Зарегистрировано в ЭДО МФТИ от 01.09.2025 № 3897-1\n\nПриложение к приказу №")
	if m.Number != "3897-1" {
		t.Fatalf("EDO requisites not found: %q", m.Number)
	}
}
