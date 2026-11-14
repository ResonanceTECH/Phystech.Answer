package mlclient

import "testing"

func TestNormalizeCompactSteps(t *testing.T) {
	sources := []Source{{Ref: 1, Title: "A"}, {Ref: 2, Title: "B", ChunkID: "c2"}}
	steps := []Step{
		{Text: "Подать служебную записку [1][2]", Refs: []int{1, 2, 9}},
		{Order: 2, Title: "Structured", Description: "keep as is"},
	}
	normalizeCompactSteps(steps, sources)

	if steps[0].Description != "Подать служебную записку [1][2]" || steps[0].Title == "" {
		t.Fatalf("compact step not mapped: %+v", steps[0])
	}
	if len(steps[0].CitationIDs) != 2 || steps[0].CitationIDs[0] != "ref:1" || steps[0].CitationIDs[1] != "c2" {
		t.Fatalf("citation ids: %v", steps[0].CitationIDs)
	}
	if sources[0].ChunkID != "ref:1" {
		t.Fatalf("source chunk id not set: %q", sources[0].ChunkID)
	}
	if steps[1].Description != "keep as is" || len(steps[1].CitationIDs) != 0 {
		t.Fatalf("structured step changed: %+v", steps[1])
	}
}
