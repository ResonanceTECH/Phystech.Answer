package rag

import (
	"context"
	"log"
	"sort"
	"sync"

	"phystech-answer/ml/internal/store"
)

// decisionMin: P(fragment answers the question) of the best candidate
// below which the base is considered to have no answer. The decision
// model is calibrated, so 0.5 is the natural "more likely yes than no".
const decisionMin = 0.5

// decisionKeep: weaker candidates are dropped from the generator context
// (at least two are always kept for context).
const decisionKeep = 0.3

const decisionParallel = 4

// rerankDecision asks the decision model, for every candidate, whether the
// fragment answers the question, and orders candidates by that probability.
// For a composite question the score is the maximum over its subqueries: a
// fragment only needs to answer one independent part to remain in context.
func (s *Service) rerankDecision(ctx context.Context, queries []string, pool []store.ScoredChunk) ([]store.ScoredChunk, bool) {
	if len(queries) == 0 {
		return nil, false
	}
	probs := make([]float64, len(pool))
	errs := make([]error, len(pool))
	sem := make(chan struct{}, decisionParallel)
	var wg sync.WaitGroup
	for i, sc := range pool {
		wg.Add(1)
		go func(i int, sc store.ScoredChunk) {
			defer wg.Done()
			for _, q := range queries {
				sem <- struct{}{}
				p, err := s.decider.Relevance(ctx, q, sc.Chunk.Context, sc.Chunk.Text)
				<-sem
				if err != nil {
					errs[i] = err
					continue
				}
				errs[i] = nil
				if p > probs[i] {
					probs[i] = p
				}
			}
		}(i, sc)
	}
	wg.Wait()
	failed := 0
	for _, err := range errs {
		if err != nil {
			failed++
		}
	}
	if failed > len(pool)/2 {
		log.Printf("rag: decision rerank failed (%d/%d): %v", failed, len(pool), errs[0])
		return nil, false
	}

	type item struct {
		sc   store.ScoredChunk
		p    float64
		rank int
	}
	items := make([]item, len(pool))
	for i, sc := range pool {
		items[i] = item{sc: sc, p: probs[i], rank: i}
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].p != items[j].p {
			return items[i].p > items[j].p
		}
		return items[i].rank < items[j].rank
	})
	out := make([]store.ScoredChunk, 0, len(items))
	for _, it := range items {
		if it.p < decisionKeep && len(out) >= 2 {
			break
		}
		c := it.sc
		c.Score = it.p
		out = append(out, c)
	}
	return out, true
}
