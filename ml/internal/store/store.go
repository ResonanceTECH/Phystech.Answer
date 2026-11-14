// Package store keeps the chunk index in memory and persists it to a JSON
// snapshot. It is deliberately simple: the knowledge base is small enough
// (tens of documents) that brute-force cosine search is instant. Swap this
// for pgvector/Postgres when the corpus grows.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"

	"phystech-answer/ml/internal/chunk"
	"phystech-answer/ml/internal/model"
)

// Embedder computes vectors for texts; implemented by the embed client.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// ScoredChunk is a chunk with its relevance score.
type ScoredChunk struct {
	Chunk model.Chunk
	Score float64
}

var ErrNotFound = errors.New("document not found")

type Store struct {
	mu        sync.RWMutex
	path      string
	docs      map[string]model.Document // docID -> document (metadata + content)
	chunks    map[string]model.Chunk    // chunkID -> chunk
	docChunks map[string][]string       // docID -> chunk IDs
	norms     map[string]float64        // chunkID -> |vector|
}

type snapshot struct {
	Docs   map[string]model.Document `json:"docs"`
	Chunks []model.Chunk             `json:"chunks"`
}

func New(path string) *Store {
	s := &Store{
		path:      path,
		docs:      map[string]model.Document{},
		chunks:    map[string]model.Chunk{},
		docChunks: map[string][]string{},
		norms:     map[string]float64{},
	}
	s.load()
	return s
}

// Stats returns index sizes for health checks.
func (s *Store) Stats() (docs, chunks int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs), len(s.chunks)
}

// ListDocuments returns indexed document metadata (without content).
func (s *Store) ListDocuments() []model.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Document, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, model.Document{ID: d.ID, Title: d.Title, Tags: d.Tags})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// Upsert indexes a document: chunks it, embeds the chunks and replaces any
// previous chunks of the same document.
func (s *Store) Upsert(ctx context.Context, doc model.Document, emb Embedder) error {
	chunks := chunk.Split(doc)
	if len(chunks) == 0 {
		return nil
	}

	if emb != nil {
		texts := make([]string, len(chunks))
		for i, c := range chunks {
			texts[i] = c.Text
		}
		vecs, err := emb.Embed(ctx, texts)
		if err != nil {
			log.Printf("store: embed %q: %v (chunks kept without vectors)", doc.Title, err)
		} else {
			for i := range chunks {
				if i < len(vecs) {
					chunks[i].Vector = vecs[i]
				}
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeChunksLocked(doc.ID)
	s.docs[doc.ID] = doc
	ids := make([]string, 0, len(chunks))
	for i := range chunks {
		chunks[i].ID = doc.ID + "#" + strconv.Itoa(i)
		if n := vectorNorm(chunks[i].Vector); n > 0 {
			s.norms[chunks[i].ID] = n
		}
		s.chunks[chunks[i].ID] = chunks[i]
		ids = append(ids, chunks[i].ID)
	}
	s.docChunks[doc.ID] = ids
	return s.saveLocked()
}

// Delete removes a document and all its chunks.
func (s *Store) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.docs[id]; !ok {
		return ErrNotFound
	}
	s.removeChunksLocked(id)
	delete(s.docs, id)
	return s.saveLocked()
}

// ReplaceAll rebuilds the whole index from the given documents (used by
// /api/sync). Embedding happens before the swap so readers never see a
// half-built index.
func (s *Store) ReplaceAll(ctx context.Context, docs []model.Document, emb Embedder) error {
	type prepared struct {
		doc    model.Document
		chunks []model.Chunk
	}
	prepped := make([]prepared, 0, len(docs))
	for _, d := range docs {
		cs := chunk.Split(d)
		if len(cs) == 0 {
			continue
		}
		prepped = append(prepped, prepared{doc: d, chunks: cs})
	}

	if emb != nil {
		// Embed in batches to stay within request size limits. Once the
		// embedder fails, skip the remaining calls — chunks stay vectorless
		// and remain reachable via lexical search.
		const batch = 32
		embedOK := true
		for i := 0; i < len(prepped) && embedOK; i++ {
			for j := 0; j < len(prepped[i].chunks); j += batch {
				end := j + batch
				if end > len(prepped[i].chunks) {
					end = len(prepped[i].chunks)
				}
				texts := make([]string, end-j)
				for k := j; k < end; k++ {
					texts[k-j] = prepped[i].chunks[k].Text
				}
				vecs, err := emb.Embed(ctx, texts)
				if err != nil {
					log.Printf("store: embed batch: %v (chunks kept without vectors)", err)
					embedOK = false
					break
				}
				for k := j; k < end && k-j < len(vecs); k++ {
					prepped[i].chunks[k].Vector = vecs[k-j]
				}
			}
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs = map[string]model.Document{}
	s.chunks = map[string]model.Chunk{}
	s.docChunks = map[string][]string{}
	s.norms = map[string]float64{}
	for _, p := range prepped {
		ids := make([]string, 0, len(p.chunks))
		for i := range p.chunks {
			p.chunks[i].ID = p.doc.ID + "#" + strconv.Itoa(i)
			if n := vectorNorm(p.chunks[i].Vector); n > 0 {
				s.norms[p.chunks[i].ID] = n
			}
			s.chunks[p.chunks[i].ID] = p.chunks[i]
			ids = append(ids, p.chunks[i].ID)
		}
		s.docChunks[p.doc.ID] = ids
		s.docs[p.doc.ID] = p.doc
	}
	return s.saveLocked()
}

// SearchVector returns chunks ranked by cosine similarity to queryVec.
// Results are sorted descending; the caller applies the score threshold.
func (s *Store) SearchVector(queryVec []float32, topK int) []ScoredChunk {
	if len(queryVec) == 0 {
		return nil
	}
	qn := vectorNorm(queryVec)
	if qn == 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	scored := make([]ScoredChunk, 0, len(s.chunks))
	for _, c := range s.chunks {
		if len(c.Vector) == 0 {
			continue
		}
		scored = append(scored, ScoredChunk{Chunk: c, Score: cosine(queryVec, qn, c.Vector, s.norms[c.ID])})
	}
	return topN(scored, topK)
}

// SearchLexical ranks chunks by token overlap with the query. Used when the
// embedding service is unavailable. Scores are heuristics in ~[0,1].
func (s *Store) SearchLexical(query string, topK int) []ScoredChunk {
	terms := queryTerms(query)
	if len(terms) == 0 {
		return nil
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	scored := make([]ScoredChunk, 0, len(s.chunks))
	for _, c := range s.chunks {
		scored = append(scored, ScoredChunk{Chunk: c, Score: lexicalScore(terms, c)})
	}
	return topN(scored, topK)
}

func (s *Store) removeChunksLocked(docID string) {
	for _, id := range s.docChunks[docID] {
		delete(s.chunks, id)
		delete(s.norms, id)
	}
	delete(s.docChunks, docID)
}

func topN(scored []ScoredChunk, k int) []ScoredChunk {
	sort.Slice(scored, func(i, j int) bool { return scored[i].Score > scored[j].Score })
	// Drop zero-score tail.
	end := len(scored)
	for end > 0 && scored[end-1].Score <= 0 {
		end--
	}
	scored = scored[:end]
	if k > 0 && len(scored) > k {
		scored = scored[:k]
	}
	return scored
}

func cosine(a []float32, anorm float64, b []float32, bnorm float64) float64 {
	if bnorm == 0 || len(a) != len(b) {
		return 0
	}
	var dot float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
	}
	return dot / (anorm * bnorm)
}

func vectorNorm(v []float32) float64 {
	var sum float64
	for _, x := range v {
		sum += float64(x) * float64(x)
	}
	return math.Sqrt(sum)
}

var tokenRe = regexp.MustCompile(`[\p{L}\p{N}]+`)

// ruStopwords are frequent function words that carry no retrieval signal.
var ruStopwords = map[string]bool{
	"и": true, "в": true, "на": true, "что": true, "как": true, "для": true,
	"по": true, "от": true, "до": true, "не": true, "но": true, "а": true,
	"с": true, "со": true, "к": true, "у": true, "о": true, "об": true,
	"это": true, "тот": true, "так": true, "же": true, "ли": true, "бы": true,
	"при": true, "про": true, "без": true, "под": true, "над": true, "из": true,
	"за": true, "между": true, "через": true, "после": true, "перед": true,
	"во": true, "ко": true, "его": true, "её": true, "их": true, "он": true,
	"она": true, "они": true, "мы": true, "вы": true, "я": true, "мне": true,
	"нам": true, "вам": true, "свой": true, "своя": true, "своё": true,
	"который": true, "которая": true, "которые": true, "если": true,
	"также": true, "или": true, "уже": true, "еще": true, "ещё": true,
	"был": true, "была": true, "были": true, "есть": true, "быт": true,
}

// normalizeToken unifies ё/е spelling so "счёт" and "счет" match.
func normalizeToken(s string) string {
	return strings.ReplaceAll(s, "ё", "е")
}

// queryTerms extracts lowercase word tokens (len >= 3, stopwords removed)
// from the query.
func queryTerms(q string) []string {
	seen := map[string]bool{}
	var out []string
	for _, t := range tokenRe.FindAllString(normalizeToken(strings.ToLower(q)), -1) {
		if utf8.RuneCountInString(t) < 3 || seen[t] || ruStopwords[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

func chunkTerms(c model.Chunk) map[string]bool {
	text := c.DocTitle + " " + strings.Join(c.Tags, " ") + " " + c.Section + " " + c.Text
	set := map[string]bool{}
	for _, t := range tokenRe.FindAllString(normalizeToken(strings.ToLower(text)), -1) {
		if utf8.RuneCountInString(t) >= 3 {
			set[t] = true
		}
	}
	return set
}

// termMatch reports whether query term q matches chunk term t. Exact match
// always wins; otherwise a long common rune-prefix counts as a match so
// that Russian word forms line up ("конференцию" ~ "конференции",
// "командировка" ~ "командировки").
func termMatch(q, t string) bool {
	if q == t {
		return true
	}
	qr, tr := []rune(q), []rune(t)
	shorter := len(qr)
	if len(tr) < shorter {
		shorter = len(tr)
	}
	common := 0
	for common < shorter && qr[common] == tr[common] {
		common++
	}
	// Need a substantial shared stem: at least 5 runes and almost the whole
	// shorter word, so unrelated terms don't collide.
	return common >= 5 && common >= shorter-3
}

// synonyms maps colloquial query terms to domain vocabulary used in the
// regulations. It compensates for the vocabulary gap that embeddings would
// normally cover (e.g. "сломался компьютер" → "неисправность оборудования").
var synonyms = map[string][]string{
	"компьютер":     {"оборудование", "оргтехника", "техника", "ноутбук"},
	"ноутбук":       {"оборудование", "оргтехника", "техника"},
	"принтер":       {"оборудование", "оргтехника"},
	"сломался":      {"неисправность", "неисправен", "ремонт", "поломка"},
	"сломалась":     {"неисправность", "неисправен", "ремонт", "поломка"},
	"не работает":   {"неисправность", "ремонт"},
	"деньги":        {"средства", "финансирование", "финансы", "оплата"},
	"получить":      {"выдача", "получение", "предоставление"},
	"проект":        {"грант", "хоздоговор", "договор"},
	"грант":         {"проект", "соглашение", "грантодатель"},
	"смета":         {"бюджет", "расходы", "сметный"},
	"бюджет":        {"смета", "расходы", "бюджетный"},
	"нанять":        {"подбор", "найм", "персонал"},
	"сотрудник":     {"персонал", "кандидат", "работник"},
	"сотрудника":    {"персонал", "кандидат", "работник"},
	"вакансия":      {"подбор", "кандидат", "персонал"},
	"резюме":        {"кандидат", "подбор"},
	"командировка":  {"поездка", "командирован"},
	"командировку":  {"поездка", "командирован"},
	"конференция":   {"мероприятие", "конференций"},
	"аванс":         {"выдача", "деньги", "отчет"},
	"отчет":         {"отчетность", "авансовый"},
	"заимствование": {"перемещение", "заем", "заимствования"},
	"поддержка":     {"обращение", "заявка", "цит"},
	"обращение":     {"заявка", "поддержка", "помощь"},
	"заявка":        {"заявление", "обращение"},
	"заявление":     {"заявка", "обращение"},
	"счет":          {"флс", "финансово", "лицевой"},
	"счёт":          {"флс", "финансово", "лицевой"},
	"руководитель":  {"начальник", "подразделения"},
}

// termMatches reports whether query term q matches any chunk term,
// considering morphological variants and domain synonyms.
func termMatches(q string, terms map[string]bool) bool {
	candidates := append([]string{q}, synonyms[q]...)
	for _, cand := range candidates {
		for t := range terms {
			if termMatch(cand, t) {
				return true
			}
		}
	}
	return false
}

func lexicalScore(queryTerms []string, c model.Chunk) float64 {
	terms := chunkTerms(c)
	matched := 0
	for _, q := range queryTerms {
		if termMatches(q, terms) {
			matched++
		}
	}
	if matched == 0 {
		return 0
	}
	overlap := float64(matched) / float64(len(queryTerms))
	// Boost when the document title or tags match the query.
	titleTerms := map[string]bool{}
	titleText := normalizeToken(strings.ToLower(c.DocTitle + " " + strings.Join(c.Tags, " ")))
	for _, t := range tokenRe.FindAllString(titleText, -1) {
		if utf8.RuneCountInString(t) >= 3 {
			titleTerms[t] = true
		}
	}
	bonus := 0.0
	for _, q := range queryTerms {
		if termMatches(q, titleTerms) {
			bonus = 0.15
			break
		}
	}
	score := overlap*0.85 + bonus
	if score > 0.95 {
		score = 0.95
	}
	return score
}

func (s *Store) saveLocked() error {
	if s.path == "" {
		return nil
	}
	snap := snapshot{Docs: s.docs, Chunks: make([]model.Chunk, 0, len(s.chunks))}
	for _, c := range s.chunks {
		snap.Chunks = append(snap.Chunks, c)
	}
	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	if dir := filepath.Dir(s.path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) load() {
	if s.path == "" {
		return
	}
	data, err := os.ReadFile(s.path)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("store: read index %s: %v", s.path, err)
		}
		return
	}
	var snap snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		log.Printf("store: parse index %s: %v", s.path, err)
		return
	}
	for id, d := range snap.Docs {
		s.docs[id] = d
	}
	for _, c := range snap.Chunks {
		s.chunks[c.ID] = c
		s.docChunks[c.DocID] = append(s.docChunks[c.DocID], c.ID)
		if n := vectorNorm(c.Vector); n > 0 {
			s.norms[c.ID] = n
		}
	}
	log.Printf("store: loaded %d documents, %d chunks from %s", len(s.docs), len(s.chunks), s.path)
}
