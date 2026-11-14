package rag

import (
	"context"
	"encoding/json"
	"log"
	"regexp"
	"strings"

	"phystech-answer/ml/internal/textproc"
)

// Composite questions ("еду на конференцию, а после хочу провести семинар
// в зале") mix several procedures. A single embedding/BM25 query is pulled
// towards one of them, so the question is split into sub-queries that are
// retrieved separately and fused with RRF.

var compositeMarkers = regexp.MustCompile(`(?i)(а после|после этого|после возвращения|потом|затем|а также|кроме того|и как|и какой|и какие|и что|и куда|и кто|и когда|и где|а ещё|а еще)`)

// looksComposite is a cheap gate so simple questions skip the LLM call.
func looksComposite(q string) bool {
	words := len(textproc.Tokens(q))
	sentences := strings.Count(q, "?") + strings.Count(q, ". ")
	return words >= 9 && (compositeMarkers.MatchString(q) || sentences >= 2)
}

const decomposePrompt = `Ты помогаешь искать по базе внутренних регламентов МФТИ.
Если вопрос сотрудника состоит из нескольких независимых задач, раздели его на 2–3 коротких самостоятельных поисковых запроса — по одному на каждую задачу.
Если задача одна — верни один запрос, совпадающий по смыслу с вопросом.
Не добавляй ничего, чего нет в вопросе.
Верни ТОЛЬКО JSON: {"queries": ["...", "..."]}`

func (s *Service) decompose(ctx context.Context, q string) []string {
	raw, err := s.chat.ChatJSON(ctx, decomposePrompt, "Вопрос: "+q)
	if err != nil {
		log.Printf("rag: decompose failed: %v", err)
		return nil
	}
	var out struct {
		Queries []string `json:"queries"`
	}
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		return nil
	}
	var qs []string
	for _, sub := range out.Queries {
		sub = strings.TrimSpace(sub)
		if sub != "" && !strings.EqualFold(sub, q) && len(qs) < 3 {
			qs = append(qs, sub)
		}
	}
	if len(qs) < 2 {
		return nil // nothing to split
	}
	return qs
}
