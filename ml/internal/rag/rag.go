// Package rag implements the retrieval-augmented generation pipeline:
// hybrid retrieval (vector + lexical + RRF), cross-encoder reranking,
// grounded JSON answer generation and verification against the retrieved
// chunks.
package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"phystech-answer/ml/internal/chat"
	"phystech-answer/ml/internal/config"
	"phystech-answer/ml/internal/embed"
	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/prompts"
	"phystech-answer/ml/internal/relations"
	"phystech-answer/ml/internal/rerank"
	"phystech-answer/ml/internal/store"
)

const insufficientSummary = `В предоставленной базе регламентов не найдено подтверждённой информации по этому вопросу.

Возможные причины:
- вопрос относится к документам, которых нет в базе знаний;
- вопрос сформулирован слишком общо — попробуйте уточнить процедуру;
- в найденных документах есть только часть сведений — тогда уточните вопрос, и я покажу, что именно удалось подтвердить.`

type Service struct {
	st       *store.Store
	embedder *embed.Client // nil → lexical-only retrieval
	chat     *chat.Client  // nil → degraded excerpt answers
	reranker *rerank.Client
	prompts  *prompts.Store
	cfg      config.Config

	mu        sync.RWMutex
	relations map[string][]model.Relation // docID → extracted relations
}

func New(st *store.Store, em *embed.Client, ch *chat.Client, rr *rerank.Client, cfg config.Config) *Service {
	if cfg.TopK <= 0 {
		cfg.TopK = 8
	}
	s := &Service{
		st: st, embedder: em, chat: ch, reranker: rr, cfg: cfg,
		relations: map[string][]model.Relation{},
	}
	s.prompts = prompts.New(cfg.LLMPromptDir, cfg.PromptVersion)
	return s
}

func (s *Service) Stats() (docs, chunks int) { return s.st.Stats() }

func (s *Service) IndexDocument(ctx context.Context, doc model.Document) error {
	err := s.st.Upsert(ctx, doc, s.embedder)
	if err == nil {
		s.refreshRelations(doc)
	}
	return err
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	s.mu.Lock()
	delete(s.relations, id)
	s.mu.Unlock()
	return s.st.Delete(ctx, id)
}

func (s *Service) SyncDocuments(ctx context.Context, docs []model.Document) error {
	err := s.st.ReplaceAll(ctx, docs, s.embedder)
	if err != nil {
		return err
	}
	// Rebuild relation index from corpus metadata.
	s.mu.Lock()
	s.relations = map[string][]model.Relation{}
	for _, d := range docs {
		rels := relations.Extract(d)
		s.relations[d.ID] = relations.ResolveTargets(rels, docs)
	}
	s.mu.Unlock()
	return nil
}

func (s *Service) ListDocuments() []model.Document { return s.st.ListDocuments() }

// RelationsFor returns extracted relations of a document.
func (s *Service) RelationsFor(docID string) []model.Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.relations[docID]
}

// Relations returns the whole relation index (for sync/debug endpoints).
func (s *Service) Relations() map[string][]model.Relation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string][]model.Relation, len(s.relations))
	for k, v := range s.relations {
		out[k] = v
	}
	return out
}

func (s *Service) refreshRelations(doc model.Document) {
	rels := relations.Extract(doc)
	// Resolve against the current corpus.
	docs := s.st.ListDocuments()
	rels = relations.ResolveTargets(rels, docs)
	s.mu.Lock()
	s.relations[doc.ID] = rels
	s.mu.Unlock()
}

// ---------- ask pipeline ----------

// Ask answers a user question using retrieved regulation fragments.
// It keeps the legacy contract: a markdown-ish text answer plus sources.
// The structured variant is AskStructured.
func (s *Service) Ask(ctx context.Context, question string) (model.AskResponse, error) {
	payload, err := s.AskStructured(ctx, question)
	if err != nil {
		return model.AskResponse{}, err
	}
	return model.AskResponse{
		Answer:             renderLegacyAnswer(payload),
		Sources:            payload.Sources,
		Status:             payload.Status,
		Steps:              payload.Steps,
		Warnings:           payload.Warnings,
		MissingInformation: payload.MissingInformation,
		Conflicts:          payload.Conflicts,
	}, nil
}

// AskStructured runs the full pipeline and returns the machine-readable
// answer payload (spec §13).
func (s *Service) AskStructured(ctx context.Context, question string) (model.AnswerPayload, error) {
	start := time.Now()
	q := strings.TrimSpace(question)
	if q == "" {
		return model.AnswerPayload{}, fmt.Errorf("question is required")
	}
	out := model.AnswerPayload{Status: "error", Sources: []model.Source{}}

	if s.chat == nil && s.embedder == nil {
		out.Status = "error"
		out.Summary = "ML-модели не сконфигурированы (LLM/embeddings отключены)."
		return out, nil
	}

	// 1. Hybrid retrieval + rerank.
	cand := s.retrieveHybrid(ctx, q, s.cfg.RetrieveVectorK, s.cfg.RetrieveLexK, s.cfg.RRFK)
	if len(cand.Chunks) == 0 {
		out.Status = "not_found"
		out.Summary = insufficientSummary
		return out, nil
	}
	candN := s.cfg.RerankCandidates
	if candN <= 0 {
		candN = 20
	}
	if len(cand.Chunks) > candN {
		cand.Chunks = cand.Chunks[:candN]
	}
	relevant := s.rerankChunks(ctx, q, cand.Chunks, s.cfg.TopK)
	if len(relevant) == 0 {
		out.Status = "not_found"
		out.Summary = insufficientSummary
		return out, nil
	}

	// Score floor: rerank relevance is a logit-ish score; cosine fallback ~[0,1].
	// Keep chunks whose score clears a low bar OR always keep top-3.
	relevant = s.filterRelevant(relevant)
	if len(relevant) == 0 {
		out.Status = "not_found"
		out.Summary = insufficientSummary
		return out, nil
	}

	// 2. Generate structured answer.
	if s.chat == nil {
		return s.degradedPayload(relevant), nil
	}
	payload, err := s.generateAnswer(ctx, q, relevant)
	if err != nil {
		log.Printf("rag: generate failed, falling back to degraded: %v", err)
		return s.degradedPayload(relevant), nil
	}

	// 3. Verify citations & statuses.
	s.verifyPayload(&payload, relevant)

	// 4. Sources + debug.
	payload.Sources = s.toSources(relevant)
	payload.Debug = &model.AnswerDebug{
		Retrieved: s.toSources(cand.Chunks),
		Reranked:  s.toSources(relevant),
		Model:     s.cfg.LLMModel,
		LatencyMS: time.Since(start).Milliseconds(),
	}
	return payload, nil
}

// filterRelevant drops weak candidates but always keeps at least 3 chunks so
// the LLM sees enough context to say "not found" reliably.
func (s *Service) filterRelevant(scored []store.ScoredChunk) []store.ScoredChunk {
	if len(scored) <= 3 {
		return scored
	}
	floor := s.cfg.MinScore
	out := make([]store.ScoredChunk, 0, len(scored))
	for _, sc := range scored {
		if sc.Score >= floor {
			out = append(out, sc)
		}
	}
	if len(out) < 3 {
		out = scored[:3]
	}
	return out
}

// ---------- generation ----------

type llmAnswerJSON struct {
	Status  string `json:"status"`
	Summary string `json:"summary"`
	Steps   []struct {
		Order       int      `json:"order"`
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Responsible *string  `json:"responsible"`
		Deadline    *string  `json:"deadline"`
		Condition   *string  `json:"condition"`
		CitationIDs []string `json:"citation_ids"`
	} `json:"steps"`
	Warnings           []string `json:"warnings"`
	MissingInformation []string `json:"missing_information"`
	Conflicts          []struct {
		Description string   `json:"description"`
		ChunkIDs    []string `json:"chunk_ids"`
	} `json:"conflicts"`
}

func (s *Service) generateAnswer(ctx context.Context, question string, relevant []store.ScoredChunk) (model.AnswerPayload, error) {
	promptTpl, err := s.prompts.Load("generate_answer")
	if err != nil {
		log.Printf("rag: prompt generate_answer missing, using builtin: %v", err)
		promptTpl = defaultGeneratePrompt
	}
	user := buildUserPrompt(promptTpl, question, relevant, s.st)

	maxTokens := 2500
	res, err := s.chat.ChatEx(ctx, chat.ChatOptions{
		User:        user,
		Temperature: 0.1,
		MaxTokens:   maxTokens,
		JSONMode:    true,
	})
	if err != nil {
		return model.AnswerPayload{}, err
	}
	return parseAnswerJSON(res.Content, relevant)
}

// AnalyzeQuestion runs the question-analysis prompt (internal/v1/questions/analyze).
func (s *Service) AnalyzeQuestion(ctx context.Context, question string) (map[string]any, error) {
	if s.chat == nil {
		return nil, fmt.Errorf("chat model not configured")
	}
	tpl, err := s.prompts.Load("analyze_question")
	if err != nil {
		tpl = defaultAnalyzePrompt
	}
	res, err := s.chat.ChatEx(ctx, chat.ChatOptions{
		User:        tpl + "\n\nВопрос: " + question,
		Temperature: 0.0,
		MaxTokens:   600,
		JSONMode:    true,
	})
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(res.Content)
	if m := jsonFenceRe.FindStringSubmatch(text); m != nil {
		text = strings.TrimSpace(m[1])
	}
	if !strings.HasPrefix(text, "{") {
		if i := strings.Index(text, "{"); i >= 0 {
			if j := strings.LastIndex(text, "}"); j > i {
				text = text[i : j+1]
			}
		}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("invalid analyze JSON: %w", err)
	}
	return out, nil
}

// VerifyAnswer runs the verification prompt against a generated payload.
func (s *Service) VerifyAnswer(ctx context.Context, question string, payload model.AnswerPayload, relevant []store.ScoredChunk) (map[string]any, error) {
	if s.chat == nil {
		return nil, fmt.Errorf("chat model not configured")
	}
	tpl, err := s.prompts.Load("verify_answer")
	if err != nil {
		tpl = defaultVerifyPrompt
	}
	ctxBlock := buildUserPrompt("", question, relevant, s.st)
	ansJSON, _ := json.Marshal(payload)
	user := ctxBlock + "\n\nСгенерированный ответ (JSON):\n" + string(ansJSON)
	res, err := s.chat.ChatEx(ctx, chat.ChatOptions{
		User:        tpl + "\n\n" + user,
		Temperature: 0.0,
		MaxTokens:   900,
		JSONMode:    true,
	})
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(res.Content)
	if m := jsonFenceRe.FindStringSubmatch(text); m != nil {
		text = strings.TrimSpace(m[1])
	}
	if !strings.HasPrefix(text, "{") {
		if i := strings.Index(text, "{"); i >= 0 {
			if j := strings.LastIndex(text, "}"); j > i {
				text = text[i : j+1]
			}
		}
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(text), &out); err != nil {
		return nil, fmt.Errorf("invalid verify JSON: %w", err)
	}
	return out, nil
}

// EmbedTexts exposes the embedder for /internal/v1/embeddings.
func (s *Service) EmbedTexts(ctx context.Context, texts []string) ([][]float32, error) {
	if s.embedder == nil {
		return nil, fmt.Errorf("embedder not configured")
	}
	return s.embedder.Embed(ctx, texts)
}

// RerankTexts exposes the reranker for /internal/v1/rerank.
func (s *Service) RerankTexts(ctx context.Context, query string, docs []string, topN int) ([]map[string]any, error) {
	if s.reranker == nil {
		return nil, fmt.Errorf("reranker not configured")
	}
	scores, err := s.reranker.Rerank(ctx, query, docs)
	if err != nil {
		return nil, err
	}
	if topN > 0 && len(scores) > topN {
		scores = scores[:topN]
	}
	out := make([]map[string]any, 0, len(scores))
	for _, sc := range scores {
		out = append(out, map[string]any{
			"index":           sc.Index,
			"relevance_score": sc.RelevanceScore,
		})
	}
	return out, nil
}

// VerifyAnswerWithContext retrieves context for the question and runs the
// verification prompt against the provided answer payload.
func (s *Service) VerifyAnswerWithContext(ctx context.Context, question string, payload model.AnswerPayload) (map[string]any, error) {
	cand := s.retrieveHybrid(ctx, question, s.cfg.RetrieveVectorK, s.cfg.RetrieveLexK, s.cfg.RRFK)
	if len(cand.Chunks) > s.cfg.RerankCandidates {
		cand.Chunks = cand.Chunks[:s.cfg.RerankCandidates]
	}
	relevant := s.rerankChunks(ctx, question, cand.Chunks, s.cfg.TopK)
	return s.VerifyAnswer(ctx, question, payload, relevant)
}

// ExtractRelationsLLM runs the LLM relation extractor over document content
// chunks (internal/v1/relations/extract).
func (s *Service) ExtractRelationsLLM(ctx context.Context, doc model.Document) ([]model.Relation, error) {
	if s.chat == nil {
		return nil, fmt.Errorf("chat model not configured")
	}
	tpl, err := s.prompts.Load("extract_relations")
	if err != nil {
		tpl = defaultExtractPrompt
	}
	// Feed the document in ~6k-char windows to stay within context.
	content := doc.Content
	var all []model.Relation
	for start := 0; start < len(content); start += 6000 {
		end := start + 6000
		if end > len(content) {
			end = len(content)
		}
		user := fmt.Sprintf("Документ: «%s»\n\nТекст:\n%s", doc.Title, content[start:end])
		res, err := s.chat.ChatEx(ctx, chat.ChatOptions{
			User:        tpl + "\n\n" + user,
			Temperature: 0.0,
			MaxTokens:   1200,
			JSONMode:    true,
		})
		if err != nil {
			return nil, err
		}
		text := strings.TrimSpace(res.Content)
		if m := jsonFenceRe.FindStringSubmatch(text); m != nil {
			text = strings.TrimSpace(m[1])
		}
		var raw []struct {
			RelationType         string `json:"relation_type"`
			TargetDocumentNumber string `json:"target_document_number"`
			TargetDocumentDate   string `json:"target_document_date"`
			TargetTitleHint      string `json:"target_title_hint"`
			Evidence             string `json:"evidence"`
		}
		if err := json.Unmarshal([]byte(text), &raw); err != nil {
			log.Printf("relations: bad JSON chunk: %v", err)
			continue
		}
		for _, r := range raw {
			all = append(all, model.Relation{
				RelationType:         r.RelationType,
				TargetDocumentNumber: r.TargetDocumentNumber,
				TargetDocumentDate:   r.TargetDocumentDate,
				Evidence:             r.Evidence,
			})
		}
	}
	return all, nil
}

func buildUserPrompt(tpl, question string, relevant []store.ScoredChunk, st *store.Store) string {
	var b strings.Builder
	b.WriteString("Фрагменты нормативных документов:\n\n")
	for _, sc := range relevant {
		doc, _ := st.Doc(sc.Chunk.DocID)
		fmt.Fprintf(&b, "[chunk_%s] Документ: «%s»", sc.Chunk.ID, sc.Chunk.DocTitle)
		if doc.DocumentNumber != "" {
			fmt.Fprintf(&b, " (№ %s", doc.DocumentNumber)
			if doc.DocumentDate != "" {
				fmt.Fprintf(&b, " от %s", doc.DocumentDate)
			}
			b.WriteString(")")
		}
		if sc.Chunk.Section != "" {
			fmt.Fprintf(&b, ", пункт %s", sc.Chunk.Section)
		}
		if sc.Chunk.Page > 0 {
			fmt.Fprintf(&b, ", с. %d", sc.Chunk.Page)
		}
		if doc.LegalStatus != "" && doc.LegalStatus != "in_force" {
			fmt.Fprintf(&b, " [статус: %s]", doc.LegalStatus)
		}
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(sc.Chunk.Text))
		b.WriteString("\n\n")
	}
	user := fmt.Sprintf("%s\n\nВопрос пользователя: %s\n\nОтветь строго JSON-объектом по заданной схеме.", b.String(), question)
	return user
}

var jsonFenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)```")

func parseAnswerJSON(content string, relevant []store.ScoredChunk) (model.AnswerPayload, error) {
	text := strings.TrimSpace(content)
	if m := jsonFenceRe.FindStringSubmatch(text); m != nil {
		text = strings.TrimSpace(m[1])
	}
	// Tolerate prose around the JSON object.
	if !strings.HasPrefix(text, "{") {
		if i := strings.Index(text, "{"); i >= 0 {
			if j := strings.LastIndex(text, "}"); j > i {
				text = text[i : j+1]
			}
		}
	}
	var raw llmAnswerJSON
	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return model.AnswerPayload{}, fmt.Errorf("invalid answer JSON: %w", err)
	}

	payload := model.AnswerPayload{
		Status:             normalizeStatus(raw.Status),
		Summary:            strings.TrimSpace(raw.Summary),
		Warnings:           raw.Warnings,
		MissingInformation: raw.MissingInformation,
		Sources:            []model.Source{},
	}
	if payload.Warnings == nil {
		payload.Warnings = []string{}
	}
	if payload.MissingInformation == nil {
		payload.MissingInformation = []string{}
	}

	// Map LLM citation ids ("chunk_…" or "[n]") to real chunk ids.
	valid := validChunkIDs(relevant)
	for _, stp := range raw.Steps {
		step := model.AnswerStep{
			Order:       stp.Order,
			Title:       strings.TrimSpace(stp.Title),
			Description: strings.TrimSpace(stp.Description),
			Responsible: stp.Responsible,
			Deadline:    stp.Deadline,
			Condition:   stp.Condition,
			CitationIDs: []string{},
		}
		for _, cid := range stp.CitationIDs {
			if real, ok := resolveCitationID(cid, valid); ok {
				step.CitationIDs = append(step.CitationIDs, real)
			}
		}
		payload.Steps = append(payload.Steps, step)
	}
	// Re-number steps sequentially — LLM order may be off.
	sort.SliceStable(payload.Steps, func(i, j int) bool { return payload.Steps[i].Order < payload.Steps[j].Order })
	for i := range payload.Steps {
		payload.Steps[i].Order = i + 1
	}

	for _, c := range raw.Conflicts {
		conf := model.Conflict{Description: strings.TrimSpace(c.Description), ChunkIDs: []string{}}
		for _, cid := range c.ChunkIDs {
			if real, ok := resolveCitationID(cid, valid); ok {
				conf.ChunkIDs = append(conf.ChunkIDs, real)
			}
		}
		payload.Conflicts = append(payload.Conflicts, conf)
	}
	return payload, nil
}

func normalizeStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "answered", "ok", "done":
		return "answered"
	case "partial":
		return "partial"
	case "clarification_required", "clarification", "needs_clarification":
		return "clarification_required"
	case "not_found", "insufficient", "no_info":
		return "not_found"
	case "conflict", "contradiction":
		return "conflict"
	default:
		return "answered"
	}
}

func validChunkIDs(relevant []store.ScoredChunk) map[string]string {
	// Maps any citation alias ("1", "[1]", "chunk_x") → real chunk id.
	m := map[string]string{}
	for i, sc := range relevant {
		m[sc.Chunk.ID] = sc.Chunk.ID
		m[fmt.Sprintf("%d", i+1)] = sc.Chunk.ID
		m[fmt.Sprintf("[%d]", i+1)] = sc.Chunk.ID
	}
	return m
}

func resolveCitationID(cid string, valid map[string]string) (string, bool) {
	cid = strings.TrimSpace(cid)
	cid = strings.TrimPrefix(cid, "[")
	cid = strings.TrimSuffix(cid, "]")
	if real, ok := valid[cid]; ok {
		return real, true
	}
	return "", false
}

// ---------- verification ----------

// verifyPayload enforces grounding: statuses consistent with evidence,
// citation references valid, contradictions surfaced.
func (s *Service) verifyPayload(p *model.AnswerPayload, relevant []store.ScoredChunk) {
	byID := map[string]store.ScoredChunk{}
	for _, sc := range relevant {
		byID[sc.Chunk.ID] = sc
	}

	// Drop citation ids that don't resolve (already filtered at parse, but be safe).
	keptSteps := p.Steps[:0]
	uncited := 0
	for _, stp := range p.Steps {
		ids := stp.CitationIDs[:0]
		for _, cid := range stp.CitationIDs {
			if _, ok := byID[cid]; ok {
				ids = append(ids, cid)
			}
		}
		stp.CitationIDs = ids
		if len(ids) == 0 {
			uncited++
		}
		keptSteps = append(keptSteps, stp)
	}
	p.Steps = keptSteps

	// Status coherence.
	switch {
	case p.Status == "answered" && uncited > 0:
		// Some steps lack citations → downgrade to partial with a warning.
		p.Status = "partial"
		p.Warnings = append(p.Warnings, fmt.Sprintf("%d шаг(ов) не подтверждены прямыми цитатами и требуют проверки.", uncited))
	case p.Status == "answered" && len(p.Steps) == 0 && p.Summary == "":
		p.Status = "not_found"
		p.Summary = insufficientSummary
	case len(p.Conflicts) > 0 && p.Status == "answered":
		p.Status = "conflict"
	}

	// Repealed-document warnings: if sources include docs marked repealed and
	// the payload didn't already warn, add one.
	for _, src := range s.toSources(relevant) {
		if src.LegalStatus == "repealed" || src.LegalStatus == "superseded" {
			p.Warnings = append(p.Warnings,
				fmt.Sprintf("Источник «%s» имеет статус «%s» — проверьте актуальную редакцию.", src.Title, src.LegalStatus))
			break
		}
	}
}

// ---------- degraded path ----------

func (s *Service) degradedPayload(relevant []store.ScoredChunk) model.AnswerPayload {
	p := model.AnswerPayload{
		Status:  "partial",
		Summary: "Языковая модель недоступна — ниже релевантные фрагменты регламентов. Проверьте их самостоятельно.",
		Warnings: []string{
			"Ответ сформирован без генерации LLM (degraded mode).",
		},
		MissingInformation: []string{},
		Conflicts:          []model.Conflict{},
		Sources:            []model.Source{},
	}
	for i, sc := range relevant {
		doc, _ := s.st.Doc(sc.Chunk.DocID)
		excerpt := strings.TrimSpace(sc.Chunk.Text)
		if r := []rune(excerpt); len(r) > 600 {
			excerpt = string(r[:600]) + "…"
		}
		p.Sources = append(p.Sources, model.Source{
			DocumentID: sc.Chunk.DocID, ChunkID: sc.Chunk.ID, Title: sc.Chunk.DocTitle,
			Section: sc.Chunk.Section, Page: sc.Chunk.Page,
			DocumentNumber: doc.DocumentNumber, LegalStatus: doc.LegalStatus,
			Excerpt: excerpt, Score: sc.Score,
		})
		_ = i
	}
	return p
}

// renderLegacyAnswer converts the structured payload into the text format
// the current backend parses (numbered lines + [n] citation markers).
func renderLegacyAnswer(p model.AnswerPayload) string {
	if p.Status == "not_found" && p.Summary == "" {
		return insufficientSummary
	}
	var b strings.Builder
	b.WriteString(p.Summary)
	if len(p.Steps) > 0 {
		b.WriteString("\n\nПорядок действий:\n")
		for _, st := range p.Steps {
			fmt.Fprintf(&b, "%d. %s", st.Order, st.Title)
			if st.Description != "" && st.Description != st.Title {
				fmt.Fprintf(&b, " — %s", st.Description)
			}
			if len(st.CitationIDs) > 0 {
				// Map real chunk ids to 1-based source positions.
				b.WriteString(" [")
				first := true
				for _, cid := range st.CitationIDs {
					if n := citationMark(cid, p.Sources); n > 0 {
						if !first {
							b.WriteString(", ")
						}
						fmt.Fprintf(&b, "%d", n)
						first = false
					}
				}
				b.WriteString("]")
			}
			b.WriteString("\n")
		}
	}
	if len(p.Warnings) > 0 {
		b.WriteString("\nПредупреждения:\n")
		for _, w := range p.Warnings {
			fmt.Fprintf(&b, "- %s\n", w)
		}
	}
	if len(p.MissingInformation) > 0 {
		b.WriteString("\nНе хватает информации:\n")
		for _, m := range p.MissingInformation {
			fmt.Fprintf(&b, "- %s\n", m)
		}
	}
	return strings.TrimSpace(b.String())
}

// citationMark returns the 1-based index of the source carrying chunkID,
// or 0 when the chunk isn't among the cited sources.
func citationMark(chunkID string, sources []model.Source) int {
	for i, s := range sources {
		if s.ChunkID == chunkID {
			return i + 1
		}
	}
	return 0
}
