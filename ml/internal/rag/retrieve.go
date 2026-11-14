package rag

import (
	"context"
	"log"
	"sort"
	"strings"

	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/rerank"
	"phystech-answer/ml/internal/store"
)

// retrievalResult is the merged candidate list before reranking.
type retrievalResult struct {
	Chunks []store.ScoredChunk
	// per-candidate diagnostics for benchmarks
	vectorRanks map[string]int
	lexRanks    map[string]int
	rrfScores   map[string]float64
}

// retrieveHybrid runs vector + lexical search and merges rankings with
// Reciprocal Rank Fusion. When the embedder is unavailable it degrades to
// pure lexical search.
func (s *Service) retrieveHybrid(ctx context.Context, question string, vectorK, lexK, rrfK int) retrievalResult {
	res := retrievalResult{
		vectorRanks: map[string]int{},
		lexRanks:    map[string]int{},
		rrfScores:   map[string]float64{},
	}

	byID := map[string]store.ScoredChunk{}
	add := func(sc store.ScoredChunk) {
		if _, ok := byID[sc.Chunk.ID]; !ok {
			byID[sc.Chunk.ID] = sc
		}
	}

	// --- vector leg ---
	if s.embedder != nil {
		vecs, err := s.embedder.Embed(ctx, []string{question})
		if err == nil && len(vecs) > 0 && len(vecs[0]) > 0 {
			vec := s.st.SearchVector(vecs[0], vectorK)
			for i, sc := range vec {
				res.vectorRanks[sc.Chunk.ID] = i + 1
				add(sc)
			}
		} else if err != nil {
			log.Printf("rag: query embed failed, lexical only: %v", err)
		}
	}

	// --- lexical leg ---
	lex := s.st.SearchLexical(question, lexK)
	for i, sc := range lex {
		res.lexRanks[sc.Chunk.ID] = i + 1
		add(sc)
	}

	// --- RRF merge ---
	if rrfK <= 0 {
		rrfK = 60
	}
	for id, sc := range byID {
		score := 0.0
		if r, ok := res.vectorRanks[id]; ok {
			score += 1.0 / float64(rrfK+r)
		}
		if r, ok := res.lexRanks[id]; ok {
			score += 1.0 / float64(rrfK+r)
		}
		res.rrfScores[id] = score
		sc.Score = score
		res.Chunks = append(res.Chunks, sc)
	}
	sort.Slice(res.Chunks, func(i, j int) bool {
		return res.rrfScores[res.Chunks[i].Chunk.ID] > res.rrfScores[res.Chunks[j].Chunk.ID]
	})
	return res
}

// rerankChunks scores candidates with the cross-encoder reranker and returns
// the top-n in reranker order. Falls back to RRF order on failure.
func (s *Service) rerankChunks(ctx context.Context, question string, cands []store.ScoredChunk, topN int) []store.ScoredChunk {
	if s.reranker == nil || len(cands) == 0 {
		if topN > 0 && len(cands) > topN {
			return cands[:topN]
		}
		return cands
	}
	docs := make([]string, len(cands))
	for i, c := range cands {
		docs[i] = chunkForRerank(c)
	}
	scores, err := s.reranker.Rerank(ctx, question, docs)
	if err != nil {
		log.Printf("rag: rerank failed, keeping RRF order: %v", err)
		if topN > 0 && len(cands) > topN {
			return cands[:topN]
		}
		return cands
	}
	// scores reference original indexes; sort desc by relevance.
	order := append([]rerank.Score(nil), scores...)
	sort.Slice(order, func(i, j int) bool { return order[i].RelevanceScore > order[j].RelevanceScore })
	out := make([]store.ScoredChunk, 0, topN)
	for _, sc := range order {
		if sc.Index < 0 || sc.Index >= len(cands) {
			continue
		}
		c := cands[sc.Index]
		c.Score = sc.RelevanceScore
		out = append(out, c)
		if topN > 0 && len(out) >= topN {
			break
		}
	}
	return out
}

func chunkForRerank(c store.ScoredChunk) string {
	var b strings.Builder
	b.WriteString("Документ: ")
	b.WriteString(c.Chunk.DocTitle)
	if c.Chunk.Section != "" {
		b.WriteString(", пункт ")
		b.WriteString(c.Chunk.Section)
	}
	if c.Chunk.Page > 0 {
		b.WriteString(", с. ")
		b.WriteString(strings.TrimSpace(strings.Repeat(" ", 0)))
		b.WriteString(itoa(c.Chunk.Page))
	}
	b.WriteString("\n")
	b.WriteString(strings.TrimSpace(c.Chunk.Text))
	return b.String()
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// toSources renders scored chunks into answer citations, enriched with
// document metadata (number, legal status, page).
func (s *Service) toSources(scored []store.ScoredChunk) []model.Source {
	sources := make([]model.Source, 0, len(scored))
	for _, sc := range scored {
		doc, _ := s.st.Doc(sc.Chunk.DocID)
		excerpt := strings.TrimSpace(sc.Chunk.Text)
		if r := []rune(excerpt); len(r) > 500 {
			excerpt = string(r[:500]) + "…"
		}
		sources = append(sources, model.Source{
			DocumentID:     sc.Chunk.DocID,
			ChunkID:        sc.Chunk.ID,
			Title:          sc.Chunk.DocTitle,
			Section:        sc.Chunk.Section,
			Page:           sc.Chunk.Page,
			DocumentNumber: doc.DocumentNumber,
			LegalStatus:    doc.LegalStatus,
			Excerpt:        excerpt,
			Score:          sc.Score,
			RerankScore:    sc.Score,
		})
	}
	return sources
}

// chunkIndex maps a chunk id back to its scored entry for citation binding.
func chunkIndex(scored []store.ScoredChunk) map[string]int {
	m := make(map[string]int, len(scored))
	for i, sc := range scored {
		m[sc.Chunk.ID] = i + 1 // 1-based citation numbers
	}
	return m
}
