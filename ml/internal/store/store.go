// Package store keeps the chunk index in memory and persists it to a JSON
// snapshot. The knowledge base is small (tens of documents, ~1k chunks), so
// brute-force cosine search and an in-memory BM25 index are instant; swap
// for pgvector / OpenSearch when the corpus grows — the Store API stays.
//
// Indexes maintained:
//   - BM25 over stemmed SearchText (context header + chunk body);
//   - dense vectors (embedding of SearchText), cached by text hash so a
//     re-sync does not recompute unchanged chunks;
//   - amendment graph: which document changes / repeals which (by number).
package store

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
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

	"phystech-answer/ml/internal/chunk"
	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/textproc"
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

// Amendment links a newer act to the document it changes.
type Amendment struct {
	By     model.Document // the amending (newer) act
	Target string         // target document ID
}

type Store struct {
	mu        sync.RWMutex
	path      string
	docs      map[string]model.Document
	chunks    map[string]model.Chunk
	docChunks map[string][]string
	norms     map[string]float64

	// BM25
	tf     map[string]map[string]int // chunkID -> term -> freq
	dl     map[string]int            // chunkID -> length in terms
	df     map[string]int            // term -> number of chunks
	avgdl  float64
	vecTag map[string][]float32 // sha1(SearchText) -> vector (embedding cache)

	amendedBy map[string][]string // target docID -> amending docIDs
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
		vecTag:    map[string][]float32{},
	}
	s.load()
	s.rebuildLocked()
	return s
}

// Stats returns index sizes for health checks.
func (s *Store) Stats() (docs, chunks int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.docs), len(s.chunks)
}

// VectorCoverage reports how many chunks have embeddings.
func (s *Store) VectorCoverage() (withVec, total int) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.chunks {
		if len(c.Vector) > 0 {
			withVec++
		}
	}
	return withVec, len(s.chunks)
}

// ListDocuments returns indexed document metadata (without content).
func (s *Store) ListDocuments() []model.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]model.Document, 0, len(s.docs))
	for _, d := range s.docs {
		out = append(out, model.Document{ID: d.ID, Title: d.Title, Tags: d.Tags, Number: d.Number, Type: d.Type, Date: d.Date})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Title < out[j].Title })
	return out
}

// GetDocument returns a document with content.
func (s *Store) GetDocument(id string) (model.Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.docs[id]
	return d, ok
}

// DocChunks returns chunks of a document in order.
func (s *Store) DocChunks(docID string) []model.Chunk {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := s.docChunks[docID]
	out := make([]model.Chunk, 0, len(ids))
	for _, id := range ids {
		out = append(out, s.chunks[id])
	}
	return out
}

// AmendedBy returns newer acts that change the given document.
func (s *Store) AmendedBy(docID string) []model.Document {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []model.Document
	for _, id := range s.amendedBy[docID] {
		if d, ok := s.docs[id]; ok {
			out = append(out, d)
		}
	}
	return out
}

// Upsert indexes a document: chunks it, embeds the chunks and replaces any
// previous chunks of the same document.
func (s *Store) Upsert(ctx context.Context, doc model.Document, emb Embedder) error {
	chunks := chunk.Split(doc)
	s.embedChunks(ctx, chunks, emb)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.removeChunksLocked(doc.ID)
	s.putLocked(doc, chunks)
	s.rebuildLocked()
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
	s.rebuildLocked()
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
	var all []model.Chunk
	for _, d := range docs {
		cs := chunk.Split(d)
		prepped = append(prepped, prepared{doc: d, chunks: cs})
		all = append(all, cs...)
	}
	s.embedChunks(ctx, all, emb)
	// embedChunks filled `all`; copy vectors back per document.
	i := 0
	for p := range prepped {
		for c := range prepped[p].chunks {
			prepped[p].chunks[c].Vector = all[i].Vector
			i++
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.docs = map[string]model.Document{}
	s.chunks = map[string]model.Chunk{}
	s.docChunks = map[string][]string{}
	s.norms = map[string]float64{}
	for _, p := range prepped {
		s.putLocked(p.doc, p.chunks)
	}
	s.rebuildLocked()
	return s.saveLocked()
}

// embedChunks fills vectors, reusing cached embeddings for unchanged text.
func (s *Store) embedChunks(ctx context.Context, chunks []model.Chunk, emb Embedder) {
	if emb == nil || len(chunks) == 0 {
		return
	}
	var todo []int
	s.mu.RLock()
	for i := range chunks {
		if v, ok := s.vecTag[textHash(chunks[i].SearchText())]; ok {
			chunks[i].Vector = v
		} else {
			todo = append(todo, i)
		}
	}
	s.mu.RUnlock()

	const batch = 16
	for start := 0; start < len(todo); start += batch {
		end := min(start+batch, len(todo))
		texts := make([]string, 0, end-start)
		for _, idx := range todo[start:end] {
			texts = append(texts, chunks[idx].SearchText())
		}
		vecs, err := emb.Embed(ctx, texts)
		if err != nil {
			log.Printf("store: embed batch: %v (chunks kept without vectors, BM25 still works)", err)
			return
		}
		for k, idx := range todo[start:end] {
			if k < len(vecs) {
				chunks[idx].Vector = vecs[k]
			}
		}
	}
	if len(todo) > 0 {
		log.Printf("store: embedded %d chunks (%d from cache)", len(todo), len(chunks)-len(todo))
	}
}

func (s *Store) putLocked(doc model.Document, chunks []model.Chunk) {
	s.docs[doc.ID] = doc
	ids := make([]string, 0, len(chunks))
	for i := range chunks {
		chunks[i].ID = doc.ID + "#" + strconv.Itoa(i)
		s.chunks[chunks[i].ID] = chunks[i]
		ids = append(ids, chunks[i].ID)
	}
	s.docChunks[doc.ID] = ids
}

func (s *Store) removeChunksLocked(docID string) {
	for _, id := range s.docChunks[docID] {
		delete(s.chunks, id)
		delete(s.norms, id)
	}
	delete(s.docChunks, docID)
}

// rebuildLocked recomputes BM25 statistics, vector norms, the embedding
// cache and the amendment graph. O(corpus) — fine for this scale.
func (s *Store) rebuildLocked() {
	s.tf = make(map[string]map[string]int, len(s.chunks))
	s.dl = make(map[string]int, len(s.chunks))
	s.df = map[string]int{}
	s.norms = make(map[string]float64, len(s.chunks))
	total := 0
	for id, c := range s.chunks {
		terms := textproc.Terms(c.SearchText())
		freq := map[string]int{}
		for _, t := range terms {
			freq[t]++
		}
		for t := range freq {
			s.df[t]++
		}
		s.tf[id] = freq
		s.dl[id] = len(terms)
		total += len(terms)
		if len(c.Vector) > 0 {
			s.norms[id] = vectorNorm(c.Vector)
			s.vecTag[textHash(c.SearchText())] = c.Vector
		}
	}
	if len(s.chunks) > 0 {
		s.avgdl = float64(total) / float64(len(s.chunks))
	}
	s.amendedBy = buildAmendments(s.docs)
}

// ---------- search ----------

const (
	bm25K1 = 1.2
	bm25B  = 0.75
)

// SearchBM25 ranks chunks with Okapi BM25 over stemmed terms.
func (s *Store) SearchBM25(query string, topK int) []ScoredChunk {
	qTerms := uniq(textproc.Terms(query))
	if len(qTerms) == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := float64(len(s.chunks))
	scored := make([]ScoredChunk, 0, 64)
	for id, freq := range s.tf {
		score := 0.0
		for _, q := range qTerms {
			f := float64(freq[q])
			if f == 0 {
				continue
			}
			df := float64(s.df[q])
			idf := math.Log(1 + (n-df+0.5)/(df+0.5))
			score += idf * f * (bm25K1 + 1) / (f + bm25K1*(1-bm25B+bm25B*float64(s.dl[id])/s.avgdl))
		}
		if score > 0 {
			scored = append(scored, ScoredChunk{Chunk: s.chunks[id], Score: score})
		}
	}
	return topN(scored, topK)
}

// KnownTerms counts how many of the (stemmed) terms occur in the corpus.
func (s *Store) KnownTerms(terms []string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	n := 0
	for _, t := range terms {
		if s.df[t] > 0 {
			n++
		}
	}
	return n
}

// SearchVector returns chunks ranked by cosine similarity to queryVec.
func (s *Store) SearchVector(queryVec []float32, topK int) []ScoredChunk {
	qn := vectorNorm(queryVec)
	if qn == 0 {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	scored := make([]ScoredChunk, 0, len(s.chunks))
	for id, c := range s.chunks {
		if len(c.Vector) == 0 {
			continue
		}
		scored = append(scored, ScoredChunk{Chunk: c, Score: cosine(queryVec, qn, c.Vector, s.norms[id])})
	}
	return topN(scored, topK)
}

// BestChunkOf returns the chunk of docID that best matches the query
// (BM25), or the first chunk when nothing matches.
func (s *Store) BestChunkOf(docID, query string) (model.Chunk, bool) {
	for _, sc := range s.SearchBM25(query, 0) {
		if sc.Chunk.DocID == docID {
			return sc.Chunk, true
		}
	}
	cs := s.DocChunks(docID)
	if len(cs) == 0 {
		return model.Chunk{}, false
	}
	return cs[0], true
}

// ---------- amendments ----------

var amendCtx = regexp.MustCompile(`(?i)(изменени|дополнени|утратившим|утратившими|отмен|в следующей редакции|в редакции)`)

// buildAmendments detects "A changes B": A's text mentions B's number near
// an amendment keyword, and A is not older than B.
func buildAmendments(docs map[string]model.Document) map[string][]string {
	out := map[string][]string{}
	for _, a := range docs {
		text := a.Content
		for _, b := range docs {
			if a.ID == b.ID || b.Number == "" || len(b.Number) < 3 {
				continue
			}
			if a.Date != "" && b.Date != "" && a.Date < b.Date {
				continue
			}
			numRe := regexp.MustCompile(`(?:№|N)\s*` + regexp.QuoteMeta(b.Number) + `\b`)
			for _, loc := range numRe.FindAllStringIndex(text, -1) {
				from := max(0, loc[0]-300)
				to := min(len(text), loc[1]+200)
				if amendCtx.MatchString(text[from:to]) {
					out[b.ID] = append(out[b.ID], a.ID)
					break
				}
			}
		}
	}
	return out
}

// ---------- helpers ----------

func topN(scored []ScoredChunk, k int) []ScoredChunk {
	sort.Slice(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		return scored[i].Chunk.ID < scored[j].Chunk.ID
	})
	if k > 0 && len(scored) > k {
		scored = scored[:k]
	}
	return scored
}

func uniq(xs []string) []string {
	seen := map[string]bool{}
	out := xs[:0]
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}

func textHash(s string) string {
	h := sha1.Sum([]byte(s))
	return hex.EncodeToString(h[:])
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

// ---------- persistence ----------

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
	sort.Slice(snap.Chunks, func(i, j int) bool { return chunkOrder(snap.Chunks[i].ID) < chunkOrder(snap.Chunks[j].ID) })
	for _, c := range snap.Chunks {
		s.chunks[c.ID] = c
		s.docChunks[c.DocID] = append(s.docChunks[c.DocID], c.ID)
	}
	log.Printf("store: loaded %d documents, %d chunks from %s", len(s.docs), len(s.chunks), s.path)
}

// chunkOrder sorts "doc#12" after "doc#2".
func chunkOrder(id string) string {
	doc, n, _ := strings.Cut(id, "#")
	i, _ := strconv.Atoi(n)
	return doc + "#" + strconv.FormatInt(int64(i)+1_000_000, 10)
}
