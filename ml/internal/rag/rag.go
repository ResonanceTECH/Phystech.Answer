// Package rag implements the retrieval-augmented generation pipeline:
// chunk retrieval (vector or lexical) followed by grounded answer
// generation with citations.
package rag

import (
	"context"
	"fmt"
	"log"
	"strings"

	"phystech-answer/ml/internal/chat"
	"phystech-answer/ml/internal/embed"
	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/store"
)

const systemPrompt = `Ты — «Физтех.Ответ», ассистент сотрудника МФТИ по внутренним регламентам и административным процедурам.

Правила:
- Отвечай только на основе приведённых фрагментов документов. Не используй внешние знания.
- Если в фрагментах нет ответа на вопрос или информации недостаточно — прямо скажи об этом и укажи, каких именно сведений не хватает. Не додумывай сроки, ответственных, номера приказов и требования.
- Если вопрос описывает процедуру, оформи ответ как пошаговую инструкцию: нумерованные шаги, необходимые документы, ответственные участники, сроки.
- После каждого существенного утверждения указывай источник в квадратных скобках, например [1] или [2].
- Пиши по-русски, кратко и по делу, сохраняя точные формулировки требований из документов.
- Фрагменты могут быть из разных документов — учитывай все релевантные.
- Если фрагменты противоречат друг другу, покажи оба положения и предупреди о неоднозначности.`

const insufficientAnswer = `В предоставленной базе регламентов не найдено подтверждённой информации по этому вопросу.

Возможные причины:
- вопрос относится к документам, которых нет в базе знаний;
- вопрос сформулирован слишком общо — попробуйте уточнить процедуру (например, командировка по России или за границу, тип проекта для сметы);
- в найденных документах есть только часть сведений — тогда уточните вопрос, и я покажу, что именно удалось подтвердить.

Если сомневаетесь, обратитесь в административную службу университета.`

type Service struct {
	st       *store.Store
	embedder *embed.Client // nil → lexical-only retrieval
	chat     *chat.Client  // nil → degraded excerpt answers
	topK     int
	minScore float64
}

func New(st *store.Store, em *embed.Client, ch *chat.Client, topK int, minScore float64) *Service {
	if topK <= 0 {
		topK = 8
	}
	return &Service{st: st, embedder: em, chat: ch, topK: topK, minScore: minScore}
}

func (s *Service) Stats() (docs, chunks int) { return s.st.Stats() }

func (s *Service) IndexDocument(ctx context.Context, doc model.Document) error {
	return s.st.Upsert(ctx, doc, s.embedder)
}

func (s *Service) DeleteDocument(ctx context.Context, id string) error {
	return s.st.Delete(ctx, id)
}

func (s *Service) SyncDocuments(ctx context.Context, docs []model.Document) error {
	return s.st.ReplaceAll(ctx, docs, s.embedder)
}

func (s *Service) ListDocuments() []model.Document { return s.st.ListDocuments() }

// Ask answers a user question using retrieved regulation fragments.
func (s *Service) Ask(ctx context.Context, question string) (model.AskResponse, error) {
	q := strings.TrimSpace(question)
	if q == "" {
		return model.AskResponse{}, fmt.Errorf("question is required")
	}

	scored := s.retrieve(ctx, q)
	if len(scored) == 0 {
		return model.AskResponse{Answer: insufficientAnswer, Sources: []model.Source{}}, nil
	}

	relevant := make([]store.ScoredChunk, 0, len(scored))
	for _, sc := range scored {
		if sc.Score >= s.minScore {
			relevant = append(relevant, sc)
		}
	}
	if len(relevant) == 0 {
		return model.AskResponse{Answer: insufficientAnswer, Sources: []model.Source{}}, nil
	}

	if s.chat != nil {
		answer, err := s.answerWithLLM(ctx, q, relevant)
		if err == nil && answer != "" {
			return model.AskResponse{Answer: answer, Sources: toSources(relevant)}, nil
		}
		log.Printf("rag: chat failed, falling back to excerpts: %v", err)
	}
	return model.AskResponse{Answer: degradedAnswer(relevant), Sources: toSources(relevant)}, nil
}

// retrieve runs vector search when embeddings are available, lexical
// search otherwise. Vector search can return nothing when the index was
// built while the embedder was down — fall back to lexical in that case.
func (s *Service) retrieve(ctx context.Context, q string) []store.ScoredChunk {
	if s.embedder != nil {
		vecs, err := s.embedder.Embed(ctx, []string{q})
		if err == nil && len(vecs) > 0 && len(vecs[0]) > 0 {
			if res := s.st.SearchVector(vecs[0], s.topK); len(res) > 0 {
				return res
			}
			log.Printf("rag: vector search empty (index may lack vectors), using lexical")
		} else {
			log.Printf("rag: query embed failed, using lexical search: %v", err)
		}
	}
	return s.st.SearchLexical(q, s.topK)
}

func (s *Service) answerWithLLM(ctx context.Context, question string, relevant []store.ScoredChunk) (string, error) {
	var b strings.Builder
	for i, sc := range relevant {
		fmt.Fprintf(&b, "[%d] Документ: «%s»", i+1, sc.Chunk.DocTitle)
		if sc.Chunk.Section != "" {
			fmt.Fprintf(&b, ", пункт %s", sc.Chunk.Section)
		}
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(sc.Chunk.Text))
		b.WriteString("\n\n")
	}
	user := fmt.Sprintf("Фрагменты документов:\n\n%sВопрос пользователя: %s\n\nОтветь на вопрос, опираясь только на фрагменты выше.", b.String(), question)
	return s.chat.Chat(ctx, systemPrompt, user)
}

// degradedAnswer is used when no chat model is reachable: it returns the
// retrieved excerpts so the user still gets grounded material.
func degradedAnswer(relevant []store.ScoredChunk) string {
	var b strings.Builder
	b.WriteString("Языковая модель недоступна, поэтому ниже — релевантные фрагменты регламентов. Проверьте их самостоятельно:\n")
	for i, sc := range relevant {
		fmt.Fprintf(&b, "\n[%d] «%s»", i+1, sc.Chunk.DocTitle)
		if sc.Chunk.Section != "" {
			fmt.Fprintf(&b, ", пункт %s", sc.Chunk.Section)
		}
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(sc.Chunk.Text))
	}
	return b.String()
}

func toSources(relevant []store.ScoredChunk) []model.Source {
	sources := make([]model.Source, 0, len(relevant))
	for _, sc := range relevant {
		excerpt := strings.TrimSpace(sc.Chunk.Text)
		if r := []rune(excerpt); len(r) > 400 {
			excerpt = string(r[:400]) + "…"
		}
		sources = append(sources, model.Source{
			DocumentID: sc.Chunk.DocID,
			Title:      sc.Chunk.DocTitle,
			Section:    sc.Chunk.Section,
			Excerpt:    excerpt,
			Score:      sc.Score,
		})
	}
	return sources
}
