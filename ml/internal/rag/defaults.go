package rag

// Builtin prompt fallbacks — used only when prompts/<name>_v1.txt is missing.
// The canonical prompts live in ml/prompts/.

const defaultGeneratePrompt = `Сформируй ответ на вопрос пользователя СТРОГО на основе приведённых фрагментов нормативных документов.

Правила:
- Не используй внешние знания. Не придумывай сроки, ответственных, номера приказов, пункты.
- Каждый шаг ссылайся на citation_ids фрагментов вида "chunk_...".
- Недостаточно данных → status "partial" + missing_information.
- Нет ответа во фрагментах → status "not_found".
- Противоречия → status "conflict" + conflicts.
- Ответ на русском, строго один JSON-объект:
{"status":"answered|partial|clarification_required|not_found|conflict","summary":"...","steps":[{"order":1,"title":"...","description":"...","responsible":null,"deadline":null,"condition":null,"citation_ids":["chunk_..."]}],"warnings":[],"missing_information":[],"conflicts":[{"description":"...","chunk_ids":["chunk_..."]}]}`

const defaultAnalyzePrompt = `Проанализируй вопрос к базе регламентов МФТИ. Верни JSON: {"intent":"procedure|fact_lookup|document_status|multi_part|out_of_scope|unclear","domain":"...","needs_clarification":false,"missing_info":[],"subquestions":[],"key_terms":[]}`

const defaultVerifyPrompt = `Проверь ответ на соответствие фрагментам. Верни JSON: {"verdict":"pass|needs_revision|reject","unsupported_claims":[],"wrong_citations":[],"missing_conditions":[],"suggested_status":"answered|partial|not_found|conflict|clarification_required","notes":"..."}`

const defaultExtractPrompt = `Извлеки нормативные связи из текста документа. Верни JSON-массив: [{"relation_type":"approves|amends|repeals|replaces|supplements|references|has_attachment","target_document_number":"...","target_document_date":"YYYY-MM-DD","target_title_hint":"...","evidence":"..."}]`
