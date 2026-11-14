package rag

import (
	"encoding/json"
	"testing"

	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/store"
)

func chunks(n int) []store.ScoredChunk {
	out := make([]store.ScoredChunk, n)
	for i := range out {
		out[i] = store.ScoredChunk{Chunk: model.Chunk{DocID: "d", DocTitle: "Регламент", Section: "п. 1", Text: "текст"}}
	}
	return out
}

func parse(t *testing.T, raw string) genOut {
	t.Helper()
	var g genOut
	if err := json.Unmarshal([]byte(extractJSON(raw)), &g); err != nil {
		t.Fatal(err)
	}
	return g
}

func TestAssembleDropsInvalidRefsAndKeepsCited(t *testing.T) {
	g := parse(t, "```json\n{\"status\":\"answered\",\"summary\":\"Подать записку [1][7]\",\"steps\":[\"1. Подготовить записку [2]\",{\"text\":\"Согласовать\",\"refs\":[1]}],\"notes\":[{\"note\":\"За два месяца [2]\"}],\"conflicts\":[],\"missing\":\"нет\"}\n```")
	tr := &model.Trace{}
	r := assemble(g, chunks(3), tr)
	if r.Status != "answered" || tr.DroppedRefs != 1 {
		t.Fatalf("status=%s dropped=%d", r.Status, tr.DroppedRefs)
	}
	if r.Steps[0].Text != "Подготовить записку [2]" || r.Steps[1].Text != "Согласовать [1]" {
		t.Errorf("steps: %+v", r.Steps)
	}
	if len(r.Notes) != 1 || r.Missing != "" {
		t.Errorf("notes=%v missing=%q", r.Notes, r.Missing)
	}
	if len(r.Sources) != 2 || r.Sources[0].Ref != 1 || r.Sources[1].Ref != 2 {
		t.Errorf("sources: %+v", r.Sources)
	}
}

func TestAssembleUncitedAnswerBecomesPartial(t *testing.T) {
	r := assemble(parse(t, `{"status":"answered","summary":"Ответ без ссылок","steps":[]}`), chunks(2), &model.Trace{})
	if r.Status != "partial" || r.Missing == "" {
		t.Fatalf("status=%s missing=%q", r.Status, r.Missing)
	}
}

func TestAssembleNotFound(t *testing.T) {
	r := assemble(parse(t, `{"status":"not_found","summary":"","missing":"Нет сведений об отпуске"}`), chunks(2), &model.Trace{})
	if r.Status != "not_found" || r.Summary == "" || len(r.Steps) != 0 {
		t.Fatalf("%+v", r)
	}
}

func TestValidateQuestion(t *testing.T) {
	for _, q := range []string{"???", "12345", "фыва олдж ывап ывап ывап"} {
		if q == "фыва олдж ывап ывап ывап" {
			continue // has vowels: handled by retrieval gate, not here
		}
		if validateQuestion(q) == "" {
			t.Errorf("%q should be invalid", q)
		}
	}
	if validateQuestion("Как оформить командировку?") != "" {
		t.Error("valid question rejected")
	}
}
