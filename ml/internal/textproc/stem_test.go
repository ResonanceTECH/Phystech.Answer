package textproc

import "testing"

func TestStemGroupsWordForms(t *testing.T) {
	groups := [][]string{
		{"командировка", "командировки", "командировку", "командировкой"},
		{"сотрудник", "сотрудника", "сотрудникам", "сотрудниками"},
		{"бронирование", "бронирования", "бронированию"},
		{"смета", "сметы", "смету", "сметой"},
	}
	for _, g := range groups {
		want := Stem(g[0])
		for _, w := range g[1:] {
			if got := Stem(w); got != want {
				t.Errorf("Stem(%q)=%q, want %q (as %q)", w, got, want, g[0])
			}
		}
	}
}

func TestTermsDropsStopwordsKeepsNumbers(t *testing.T) {
	got := Terms("Как открыть ФЛС по приказу № 794-1?")
	want := map[string]bool{Stem("открыть"): true, "флс": true, Stem("приказу"): true, "794-1": true}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, g := range got {
		if !want[g] {
			t.Errorf("unexpected term %q in %v", g, got)
		}
	}
}
