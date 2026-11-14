// Package rag implements the answer pipeline:
//
//	question → validation → hybrid retrieval (BM25 ⊕ dense, RRF)
//	→ optional LLM rerank → amendment expansion (newer acts that change
//	the found ones) → evidence gate ("нет информации") → grounded
//	structured generation (JSON) → citation verification → response.
//
// Every stage degrades gracefully: no embedder → BM25 only; no chat model
// → extractive answer from the best fragments.
package rag

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"phystech-answer/ml/internal/chat"
	"phystech-answer/ml/internal/decide"
	"phystech-answer/ml/internal/embed"
	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/store"
	"phystech-answer/ml/internal/textproc"
)

// Options tune retrieval; zero values mean defaults. Exposed so the
// evaluation harness can compare configurations (ablation).
type Options struct {
	Mode       string  // bm25 | vector | hybrid (default)
	Reranker   string  // "" (off) | "llm" — chat model grades the list | "decision" — calibrated decision model
	Amendments bool    // pull in amending acts
	Decompose  bool    // split composite questions into sub-queries (LLM)
	TopK       int     // fragments passed to the generator
	Candidates int     // candidates per retriever before fusion
	MinVector  float64 // evidence gate: min cosine of the best fragment
	MinBM25    float64 // evidence gate: min BM25 of the best fragment
}

func (o Options) withDefaults() Options {
	if o.Mode == "" {
		o.Mode = "hybrid"
	}
	if o.TopK <= 0 {
		o.TopK = 6
	}
	if o.Candidates <= 0 {
		o.Candidates = 30
	}
	return o
}

type Service struct {
	st       *store.Store
	embedder *embed.Client  // nil → lexical-only retrieval
	chat     *chat.Client   // nil → extractive answers
	decider  *decide.Client // nil → no decision-model reranking
	opts     Options
}

func New(st *store.Store, em *embed.Client, ch *chat.Client, dc *decide.Client, opts Options) *Service {
	return &Service{st: st, embedder: em, chat: ch, decider: dc, opts: opts.withDefaults()}
}

func (s *Service) HasDecider() bool { return s.decider != nil }

func (s *Service) Options() Options    { return s.opts }
func (s *Service) Store() *store.Store { return s.st }
func (s *Service) HasEmbedder() bool   { return s.embedder != nil }
func (s *Service) HasChat() bool       { return s.chat != nil }
func (s *Service) Stats() (docs, chunks int) {
	return s.st.Stats()
}

func (s *Service) IndexDocument(ctx context.Context, doc model.Document) error {
	return s.st.Upsert(ctx, doc, s.storeEmbedder())
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	return s.st.Delete(ctx, id)
}

func (s *Service) SyncDocuments(ctx context.Context, docs []model.Document) error {
	return s.st.ReplaceAll(ctx, docs, s.storeEmbedder())
}

// storeEmbedder returns a true nil interface when embeddings are disabled:
// a nil *embed.Client wrapped in store.Embedder would compare != nil.
func (s *Service) storeEmbedder() store.Embedder {
	if s.embedder == nil {
		return nil
	}
	return s.embedder
}

func (s *Service) ListDocuments() []model.Document { return s.st.ListDocuments() }

const maxQuestionRunes = 2000

// strongBM25: BM25 score of the best fragment above which the question is
// treated as answerable regardless of the reranker (calibrated on the eval
// set: out-of-base questions peak at ~9.3).
const strongBM25 = 12.0

// Ask answers a question with the service's default options.
func (s *Service) Ask(ctx context.Context, question string) (model.AskResponse, error) {
	return s.AskWith(ctx, question, s.opts)
}

// AskWith answers a question with explicit options (used by evaluation).
func (s *Service) AskWith(ctx context.Context, question string, opts Options) (model.AskResponse, error) {
	opts = opts.withDefaults()
	start := time.Now()
	q := strings.TrimSpace(question)
	if q == "" {
		return model.AskResponse{}, fmt.Errorf("question is required")
	}
	if r := []rune(q); len(r) > maxQuestionRunes {
		q = string(r[:maxQuestionRunes])
	}
	if msg := validateQuestion(q); msg != "" {
		return model.AskResponse{Status: "invalid", Answer: msg, Summary: msg, Sources: []model.Source{}}, nil
	}
	// Gibberish / wrong keyboard layout: not a single word of the question
	// occurs in the knowledge base.
	// Requires 2+ words so a single unknown term ("отпуск?") stays a regular
	// "нет информации" answer.
	if terms := textproc.Terms(q); len(terms) >= 2 && s.st.KnownTerms(terms) == 0 {
		msg := "Не удалось распознать вопрос: ни одно слово не встречается в документах базы. Проверьте раскладку и сформулируйте вопрос обычными словами."
		return model.AskResponse{Status: "invalid", Answer: msg, Summary: msg, Sources: []model.Source{}}, nil
	}

	trace := &model.Trace{Mode: opts.Mode, Queries: []string{q}}
	selected, gate := s.Retrieve(ctx, q, opts, trace)
	trace.RetrievalMs = time.Since(start).Milliseconds()

	if len(selected) == 0 || !gate {
		resp := notFound(q)
		trace.LatencyMs = time.Since(start).Milliseconds()
		resp.Trace = trace
		return resp, nil
	}

	var resp model.AskResponse
	if s.chat != nil {
		genStart := time.Now()
		r, err := s.generate(ctx, q, selected, trace)
		trace.GenerationMs = time.Since(genStart).Milliseconds()
		trace.Model = s.chat.Model()
		if err == nil {
			resp = r
		} else {
			log.Printf("rag: generation failed, extractive fallback: %v", err)
			resp = extractive(selected)
		}
	} else {
		resp = extractive(selected)
	}
	trace.LatencyMs = time.Since(start).Milliseconds()
	resp.Trace = trace
	return resp, nil
}

// Retrieve returns fragments for the generator and whether the evidence
// gate passed. Exported for evaluation of retrieval quality alone.
func (s *Service) Retrieve(ctx context.Context, q string, opts Options, trace *model.Trace) ([]store.ScoredChunk, bool) {
	opts = opts.withDefaults()
	if trace == nil {
		trace = &model.Trace{}
	}

	queries := []string{q}
	if opts.Decompose && s.chat != nil && looksComposite(q) {
		if subs := s.decompose(ctx, q); len(subs) > 0 {
			queries = append(queries, subs...)
			trace.Queries = queries
		}
	}

	// Each query is searched by every enabled retriever; all ranked lists
	// are fused with RRF. The evidence gate looks at the best raw scores.
	var lists [][]store.ScoredChunk
	var bm25, dense []store.ScoredChunk
	for _, qq := range queries {
		var b, d []store.ScoredChunk
		if opts.Mode != "vector" {
			b = s.st.SearchBM25(qq, opts.Candidates)
		}
		if opts.Mode != "bm25" && s.embedder != nil {
			vecs, err := s.embedder.Embed(ctx, []string{qq})
			if err == nil && len(vecs) > 0 {
				d = s.st.SearchVector(vecs[0], opts.Candidates)
			} else {
				log.Printf("rag: query embedding failed, BM25 only: %v", err)
			}
		}
		if len(d) == 0 && opts.Mode == "vector" {
			b = s.st.SearchBM25(qq, opts.Candidates) // graceful fallback
		}
		if len(b) > 0 {
			lists = append(lists, b)
			if len(bm25) == 0 || b[0].Score > bm25[0].Score {
				bm25 = b
			}
		}
		if len(d) > 0 {
			lists = append(lists, d)
			if len(dense) == 0 || d[0].Score > dense[0].Score {
				dense = d
			}
		}
	}

	// Evidence gate on raw retriever scores (before fusion).
	gate := evidence(bm25, dense, opts)

	var fused []store.ScoredChunk
	switch len(lists) {
	case 0:
	case 1:
		fused = lists[0]
	default:
		fused = rrf(60, lists...)
	}
	switch {
	case len(dense) > 0 && len(bm25) > 0:
		trace.Mode = "hybrid"
	case len(dense) > 0:
		trace.Mode = "vector"
	default:
		trace.Mode = "bm25"
	}
	trace.Candidates = len(fused)

	pool := fused
	if len(pool) > rerankPool {
		pool = pool[:rerankPool]
	}
	// The reranker is a strong relevance judge, but a small model sometimes
	// under-scores a fragment that literally answers the question. A
	// confident lexical match overrides it: the generator still makes the
	// final "нет информации" call.
	strongLexical := len(bm25) > 0 && bm25[0].Score >= strongBM25
	if len(pool) > 1 {
		switch {
		case opts.Reranker == "decision" && s.decider != nil:
			if rr, ok := s.rerankDecision(ctx, q, pool); ok {
				pool = rr
				trace.Reranked, trace.Reranker = true, "decision"
				gate = (len(rr) > 0 && rr[0].Score >= decisionMin) || strongLexical
			}
		case opts.Reranker == "llm" && s.chat != nil:
			if rr, ok := s.rerank(ctx, q, pool); ok {
				pool = rr
				trace.Reranked, trace.Reranker = true, "llm"
				gate = (len(rr) > 0 && rr[0].Score >= rerankMin) || strongLexical
			}
		}
	}

	selected := diversify(pool, opts.TopK)
	if opts.Amendments {
		selected = s.expandAmendments(q, selected, trace)
	}
	return selected, gate
}

// rrf merges ranked lists with Reciprocal Rank Fusion.
func rrf(k float64, lists ...[]store.ScoredChunk) []store.ScoredChunk {
	score := map[string]float64{}
	byID := map[string]store.ScoredChunk{}
	for _, l := range lists {
		for rank, sc := range l {
			score[sc.Chunk.ID] += 1 / (k + float64(rank+1))
			byID[sc.Chunk.ID] = sc
		}
	}
	out := make([]store.ScoredChunk, 0, len(score))
	for id, sc := range score {
		c := byID[id]
		c.Score = sc
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].Chunk.ID < out[j].Chunk.ID
	})
	return out
}

func evidence(bm25, dense []store.ScoredChunk, o Options) bool {
	vecOK := len(dense) > 0 && dense[0].Score >= o.MinVector
	bmOK := len(bm25) > 0 && bm25[0].Score >= o.MinBM25
	if len(dense) == 0 {
		return bmOK
	}
	return vecOK || (bmOK && o.MinBM25 > 0)
}

// diversify takes the top fragments but limits each document to 3 so one
// long document does not crowd out a second relevant one.
func diversify(pool []store.ScoredChunk, k int) []store.ScoredChunk {
	perDoc := map[string]int{}
	var out []store.ScoredChunk
	for _, sc := range pool {
		if perDoc[sc.Chunk.DocID] >= 3 {
			continue
		}
		perDoc[sc.Chunk.DocID]++
		out = append(out, sc)
		if len(out) == k {
			break
		}
	}
	return out
}

// expandAmendments adds the best fragment of each newer act that changes a
// selected document, so the generator sees the current редакция.
func (s *Service) expandAmendments(q string, selected []store.ScoredChunk, trace *model.Trace) []store.ScoredChunk {
	have := map[string]bool{}
	for _, sc := range selected {
		have[sc.Chunk.DocID] = true
	}
	var extra []store.ScoredChunk
	for _, sc := range selected {
		for _, a := range s.st.AmendedBy(sc.Chunk.DocID) {
			if have[a.ID] {
				continue
			}
			c, ok := s.st.BestChunkOf(a.ID, q+" "+sc.Chunk.DocNumber)
			if !ok {
				continue
			}
			have[a.ID] = true
			extra = append(extra, store.ScoredChunk{Chunk: c, Score: 0})
			trace.Amendments = append(trace.Amendments,
				fmt.Sprintf("%s изменяет %s", docLabel(a.Title, a.Number, a.Date), docLabel(sc.Chunk.DocTitle, sc.Chunk.DocNumber, sc.Chunk.DocDate)))
		}
	}
	return append(selected, extra...)
}

// validateQuestion rejects inputs that are not questions in natural
// language (empty, gibberish, only digits/symbols).
func validateQuestion(q string) string {
	letters := 0
	for _, r := range q {
		if unicode.IsLetter(r) {
			letters++
		}
	}
	if letters < 3 {
		return "Не удалось распознать вопрос. Опишите задачу словами, например: «Как оформить командировку?»."
	}
	if len(textproc.Terms(q)) == 0 {
		return "Вопрос слишком общий. Уточните, какую процедуру или документ вы ищете."
	}
	if utf8.RuneCountInString(q) > 12 && !hasVowelWords(q) {
		return "Не удалось распознать вопрос. Проверьте раскладку клавиатуры и сформулируйте вопрос на русском языке."
	}
	return ""
}

func hasVowelWords(q string) bool {
	for _, t := range textproc.Tokens(q) {
		for _, r := range t {
			if strings.ContainsRune("аеиоуыэюяaeiouy", r) {
				return true
			}
		}
	}
	return false
}

func docLabel(title, number, date string) string {
	l := "«" + title + "»"
	if number != "" && !strings.Contains(title, number) {
		l += " № " + number
	}
	if date != "" {
		l += " от " + formatDate(date)
	}
	return l
}

// formatDate turns 2023-03-28 into 28.03.2023.
func formatDate(d string) string {
	if len(d) == 10 && d[4] == '-' && d[7] == '-' {
		return d[8:10] + "." + d[5:7] + "." + d[0:4]
	}
	return d
}

func toSource(ref int, sc store.ScoredChunk) model.Source {
	excerpt := strings.TrimSpace(sc.Chunk.Text)
	if r := []rune(excerpt); len(r) > 500 {
		excerpt = string(r[:500]) + "…"
	}
	return model.Source{
		Ref:        ref,
		DocumentID: sc.Chunk.DocID,
		Title:      sc.Chunk.DocTitle,
		Number:     sc.Chunk.DocNumber,
		Date:       sc.Chunk.DocDate,
		Section:    sc.Chunk.Section,
		Heading:    sc.Chunk.Heading,
		Excerpt:    excerpt,
		Score:      sc.Score,
	}
}

const notFoundText = "В предоставленной базе документов нет информации, достаточной для ответа на этот вопрос."

func notFound(q string) model.AskResponse {
	return model.AskResponse{
		Status:  "not_found",
		Summary: notFoundText,
		Missing: "Ни один из загруженных регламентов и приказов не содержит сведений по запросу «" + q + "».",
		Answer: notFoundText + "\n\nПопробуйте переформулировать вопрос или уточнить процедуру. " +
			"Если документ должен быть в базе — обратитесь к администратору базы знаний.",
		Sources: []model.Source{},
	}
}

// extractive builds an answer from the best fragments without an LLM.
func extractive(selected []store.ScoredChunk) model.AskResponse {
	var b strings.Builder
	b.WriteString("Языковая модель недоступна — ниже наиболее релевантные фрагменты документов:\n")
	sources := make([]model.Source, 0, len(selected))
	for i, sc := range selected {
		fmt.Fprintf(&b, "\n[%d] %s, %s\n%s\n", i+1, docLabel(sc.Chunk.DocTitle, sc.Chunk.DocNumber, sc.Chunk.DocDate), sc.Chunk.Section, strings.TrimSpace(sc.Chunk.Text))
		sources = append(sources, toSource(i+1, sc))
	}
	return model.AskResponse{Status: "partial", Summary: "Найдены релевантные фрагменты документов (без генерации ответа).", Answer: b.String(), Sources: sources}
}
