package rag

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"phystech-answer/ml/internal/model"
	"phystech-answer/ml/internal/store"
)

const systemPrompt = `Ты — «Физтех.Ответ», ассистент сотрудников МФТИ по внутренним регламентам и приказам.

Твоя задача — ответить на вопрос сотрудника СТРОГО по приведённым фрагментам документов.

Правила:
1. Используй только факты из фрагментов. Никаких общих знаний, догадок, «обычно», «как правило». Не додумывай сроки, суммы, ответственных, адреса, формы.
2. Каждое утверждение и каждый шаг сопровождай ссылкой на фрагмент в квадратных скобках: [1], [2][4]. Ссылайся только на номера фрагментов, которые реально подтверждают утверждение.
3. Если вопрос про процедуру — массив steps ОБЯЗАТЕЛЬНО должен содержать хотя бы один шаг. Дай порядок действий в правильной последовательности: кто делает, что подаёт, куда, в какой срок (если это есть во фрагментах). Не подменяй steps абзацем в summary.
4. На общий вопрос о процедуре дай базовый порядок действий из фрагментов, а частные условия и варианты перечисли в missing или notes. Не отказывайся отвечать только потому, что сотрудник не уточнил частный случай. Если во фрагментах действительно нет ответа — status = "not_found", а в missing кратко напиши, каких сведений не хватает. Если ответ есть лишь частично — status = "partial" и перечисли в missing, чего не хватает.
5. Если фрагменты из разных документов противоречат друг другу или более новый документ (по дате) изменяет более старый — опиши это в conflicts и в ответе опирайся на актуальную (более новую) редакцию, явно указав это.
6. Если фрагменты из нескольких документов дополняют друг друга — объедини их в единый непротиворечивый ответ, указав каждый источник.
7. Пиши по-русски, кратко, деловым языком, сохраняя точные формулировки сроков и требований.
8. Объём: summary — 1–3 предложения; steps — не более 8 шагов, по одному действию в шаге; notes — не более 3 самых важных условий (сроки, ограничения), не дублируй шаги.

Верни ТОЛЬКО JSON без markdown по схеме:
{
  "status": "answered" | "partial" | "not_found",
  "summary": "краткий ответ в 1–3 предложениях со ссылками [n]",
  "steps": ["Шаг со ссылками [n]", "..."],
  "notes": ["важные условия, исключения, сроки со ссылками [n]"],
  "conflicts": ["описание противоречия или изменения редакции со ссылками [n]"],
  "missing": "чего не хватает в документах (пусто, если ответ полный)"
}
steps — пустой массив, если вопрос не про процедуру.`

// genOut accepts loose shapes: small models sometimes return list items as
// objects ({"text": ...}) or a single string instead of a list.
type genOut struct {
	Status    any `json:"status"`
	Summary   any `json:"summary"`
	Steps     any `json:"steps"`
	Notes     any `json:"notes"`
	Conflicts any `json:"conflicts"`
	Missing   any `json:"missing"`
}

func (s *Service) generate(ctx context.Context, q string, selected []store.ScoredChunk, trace *model.Trace) (model.AskResponse, error) {
	var b strings.Builder
	for i, sc := range selected {
		c := sc.Chunk
		fmt.Fprintf(&b, "[%d] %s", i+1, docLabel(c.DocTitle, c.DocNumber, c.DocDate))
		if c.Heading != "" {
			fmt.Fprintf(&b, "; раздел «%s»", c.Heading)
		}
		if c.Section != "" && c.Section != c.Heading {
			fmt.Fprintf(&b, "; %s", c.Section)
		}
		b.WriteString("\n")
		b.WriteString(strings.TrimSpace(c.Text))
		b.WriteString("\n\n")
	}
	if len(trace.Amendments) > 0 {
		b.WriteString("Связи между документами (по реквизитам): ")
		b.WriteString(strings.Join(trace.Amendments, "; "))
		b.WriteString(".\n\n")
	}
	answerKind := "справочный ответ; steps может быть пустым"
	if looksProcedural(q) {
		answerKind = "процедура; steps обязателен и должен содержать последовательность действий"
	}
	user := fmt.Sprintf("Фрагменты документов:\n\n%sВопрос сотрудника: %s\nТип ответа: %s.\nДопустимые ссылки: только [1]–[%d].", b.String(), q, answerKind, len(selected))

	raw, err := s.chat.ChatJSON(ctx, systemPrompt, user)
	if err != nil {
		return model.AskResponse{}, err
	}
	var out genOut
	if err := json.Unmarshal([]byte(extractJSON(raw)), &out); err != nil {
		log.Printf("rag: unparsable model output: %.300q", raw)
		return model.AskResponse{}, fmt.Errorf("parse model JSON: %w", err)
	}
	resp := assemble(out, selected, trace)
	if looksProcedural(q) && resp.Status != "not_found" && len(resp.Steps) == 0 && resp.Summary != "" {
		resp.Steps = []model.Step{{Text: resp.Summary, Refs: refsInText(resp.Summary, len(selected))}}
		resp.Answer = render(resp)
	}
	return resp, nil
}

var refRe = regexp.MustCompile(`\[(\d{1,2})\]`)

var procedureQuestionRe = regexp.MustCompile(`(?i)(^|[.!?]\s*)(как\s|что\s+(?:нужно|надо|делать)\s|порядок\s|процедур|оформить|получить\s+доступ|подать\s+обращение|внести\s+корректиров)`)

func looksProcedural(q string) bool { return procedureQuestionRe.MatchString(strings.TrimSpace(q)) }

func refsInText(s string, maxRef int) []int {
	var refs []int
	for _, m := range refRe.FindAllStringSubmatch(s, -1) {
		ref, _ := strconv.Atoi(m[1])
		if ref >= 1 && ref <= maxRef && !containsInt(refs, ref) {
			refs = append(refs, ref)
		}
	}
	return refs
}

// assemble validates citations (drops references to non-existent
// fragments), collects cited sources and renders markdown.
func assemble(out genOut, selected []store.ScoredChunk, trace *model.Trace) model.AskResponse {
	n := len(selected)
	used := map[int]bool{}
	clean := func(s string) (string, []int) {
		var refs []int
		s = refRe.ReplaceAllStringFunc(s, func(m string) string {
			k, _ := strconv.Atoi(m[1 : len(m)-1])
			if k < 1 || k > n {
				trace.DroppedRefs++
				return ""
			}
			if !containsInt(refs, k) {
				refs = append(refs, k)
			}
			used[k] = true
			return m
		})
		return strings.TrimSpace(s), refs
	}

	resp := model.AskResponse{Status: normalizeStatus(stepText(out.Status))}
	resp.Summary, _ = clean(stepText(out.Summary))
	for _, st := range items(out.Steps) {
		if t, refs := clean(stepText(st)); t != "" {
			resp.Steps = append(resp.Steps, model.Step{Text: stripNumber(t), Refs: refs})
		}
	}
	for _, note := range items(out.Notes) {
		if t, _ := clean(stepText(note)); t != "" {
			resp.Notes = append(resp.Notes, t)
		}
	}
	for _, c := range items(out.Conflicts) {
		if t, refs := clean(stepText(c)); t != "" {
			resp.Conflicts = append(resp.Conflicts, model.Conflict{Description: t, Refs: refs})
		}
	}
	resp.Missing = strings.TrimSpace(stepText(out.Missing))
	if isEmptyMarker(resp.Missing) {
		resp.Missing = ""
	}

	if resp.Status == "not_found" {
		if resp.Summary == "" {
			resp.Summary = notFoundText
		}
		resp.Steps, resp.Notes = nil, nil
	}
	// An "answer" without a single valid citation is not grounded.
	if resp.Status != "not_found" && len(used) == 0 {
		resp.Status = "partial"
		if resp.Missing == "" {
			resp.Missing = "Ответ не удалось подтвердить ссылками на пункты документов — проверьте источники ниже."
		}
		for i := 0; i < min(3, n); i++ {
			used[i+1] = true
		}
	}

	refs := make([]int, 0, len(used))
	for k := range used {
		refs = append(refs, k)
	}
	sort.Ints(refs)
	resp.Sources = make([]model.Source, 0, len(refs))
	for _, k := range refs {
		resp.Sources = append(resp.Sources, toSource(k, selected[k-1]))
	}
	resp.Answer = render(resp)
	return resp
}

func render(r model.AskResponse) string {
	var b strings.Builder
	b.WriteString(r.Summary)
	if len(r.Steps) > 0 {
		b.WriteString("\n\n**Порядок действий:**\n")
		for i, s := range r.Steps {
			fmt.Fprintf(&b, "%d. %s\n", i+1, s.Text)
		}
	}
	if len(r.Notes) > 0 {
		b.WriteString("\n**Обратите внимание:**\n")
		for _, n := range r.Notes {
			fmt.Fprintf(&b, "- %s\n", n)
		}
	}
	if len(r.Conflicts) > 0 {
		b.WriteString("\n**Изменения и противоречия между документами:**\n")
		for _, c := range r.Conflicts {
			fmt.Fprintf(&b, "- %s\n", c.Description)
		}
	}
	if r.Missing != "" && r.Status != "answered" {
		fmt.Fprintf(&b, "\n**Не хватает сведений:** %s\n", r.Missing)
	}
	return strings.TrimSpace(b.String())
}

func normalizeStatus(s string) string {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "not_found", "notfound", "none":
		return "not_found"
	case "partial":
		return "partial"
	default:
		return "answered"
	}
}

// items normalises a JSON value into a list (single string → one item).
func items(v any) []any {
	switch t := v.(type) {
	case []any:
		return t
	case nil:
		return nil
	case string:
		if strings.TrimSpace(t) == "" {
			return nil
		}
	}
	return []any{v}
}

// stepText extracts text from a string or an object item; citation refs
// given as a separate field ({"text": "...", "refs": [1, 2]}) are appended
// as [n] markers.
func stepText(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		text := ""
		for _, k := range []string{"text", "step", "action", "description", "note", "conflict", "value"} {
			if s, ok := t[k].(string); ok {
				text = s
				break
			}
		}
		for _, k := range []string{"refs", "sources", "citations"} {
			if arr, ok := t[k].([]any); ok {
				for _, r := range arr {
					if f, ok := r.(float64); ok && !strings.Contains(text, fmt.Sprintf("[%d]", int(f))) {
						text += fmt.Sprintf(" [%d]", int(f))
					}
				}
			}
		}
		return text
	}
	return ""
}

// isEmptyMarker: models write "нет", "—", "N/A" instead of an empty string.
func isEmptyMarker(s string) bool {
	switch strings.ToLower(strings.Trim(s, " .-—")) {
	case "", "нет", "none", "n/a", "null", "ничего", "не требуется", "отсутствует":
		return true
	}
	return false
}

var leadingNum = regexp.MustCompile(`^\s*(?:шаг\s*)?\d+[.)]\s*`)

func stripNumber(s string) string { return leadingNum.ReplaceAllString(s, "") }

// extractJSON returns the outermost {...} object from a model reply that
// may be wrapped in prose or code fences.
func extractJSON(s string) string {
	start := strings.IndexByte(s, '{')
	end := strings.LastIndexByte(s, '}')
	if start < 0 || end <= start {
		return s
	}
	return s[start : end+1]
}

func containsInt(xs []int, x int) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}
