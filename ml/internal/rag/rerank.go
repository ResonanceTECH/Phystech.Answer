package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"phystech-answer/ml/internal/store"
)

// rerankMin: candidates scored below this (0–10 scale) are not considered
// evidence; if even the best one is below, the answer is "нет информации".
const rerankMin = 4.0

// rerankPool: how many fused candidates the LLM judge grades (one call).
const rerankPool = 12

const rerankPrompt = `Ты оцениваешь релевантность фрагментов внутренних документов МФТИ вопросу сотрудника.
Для каждого фрагмента поставь оценку от 0 до 10:
10 — фрагмент прямо содержит ответ или шаг процедуры;
6–8 — фрагмент содержит существенную часть ответа или важное условие;
3–5 — тема та же, но ответа нет;
0–2 — не относится к вопросу.
Верни ТОЛЬКО JSON: {"scores": [{"id": 1, "score": 7}, ...]} для всех фрагментов.`

// rerank asks the chat model to grade candidates (listwise, one call) and
// reorders them. Returns ok=false on any failure so the caller keeps the
// fused order.
func (s *Service) rerank(ctx context.Context, q string, pool []store.ScoredChunk) ([]store.ScoredChunk, bool) {
	var b strings.Builder
	for i, sc := range pool {
		text := strings.TrimSpace(sc.Chunk.Text)
		if r := []rune(text); len(r) > 450 {
			text = string(r[:450]) + "…"
		}
		fmt.Fprintf(&b, "[%d] %s\n%s\n\n", i+1, sc.Chunk.Context, text)
	}
	user := fmt.Sprintf("Вопрос: %s\n\nФрагменты:\n\n%s", q, b.String())
	raw, err := s.chat.ChatJSON(ctx, rerankPrompt, user)
	if err != nil {
		log.Printf("rag: rerank failed: %v", err)
		return nil, false
	}
	var out struct {
		Scores []struct {
			ID    int     `json:"id"`
			Score float64 `json:"score"`
		} `json:"scores"`
	}
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil || len(out.Scores) == 0 {
		log.Printf("rag: rerank parse failed: %v", err)
		return nil, false
	}
	scores := map[int]float64{}
	for _, sc := range out.Scores {
		scores[sc.ID] = sc.Score
	}
	type item struct {
		sc    store.ScoredChunk
		score float64
		rank  int
	}
	items := make([]item, len(pool))
	for i, sc := range pool {
		items[i] = item{sc: sc, score: scores[i+1], rank: i}
	}
	// Sort by rerank score; fused rank breaks ties.
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].score != items[j].score {
			return items[i].score > items[j].score
		}
		return items[i].rank < items[j].rank
	})
	res := make([]store.ScoredChunk, 0, len(items))
	for _, it := range items {
		if it.score < rerankMin && len(res) >= 2 {
			break // keep at least two for context, drop the irrelevant tail
		}
		c := it.sc
		c.Score = it.score
		res = append(res, c)
	}
	return res, true
}
