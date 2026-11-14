# ML / RAG-сервис

Микросервис «Физтех.Ответ» на Go (только стандартная библиотека):
индексирует тексты регламентов, отвечает на вопросы по ним через
гибридный поиск + cross-encoder rerank + структурированную генерацию
ответа с проверкой цитат. Внешние модели (LLM, эмбеддинги, реранкер)
подключаются по OpenAI-compatible HTTP API — llama.cpp/Ollama.

## Стек и устройство

- **Язык**: Go 1.25, **ноль внешних зависимостей** (`go.mod` — только
  `module` + `go`). HTTP — `net/http` `ServeMux`.
- **Хранение индекса**: in-memory + JSON-снапшот (`index.json`),
  атомарная запись через `.tmp` + rename. Осознанно простое решение —
  корпус ~20 документов, brute-force cosine мгновенен; при росте —
  замена на pgvector (см. комментарий в `internal/store/store.go`).
- **Клиенты моделей**: `internal/chat` (chat completions, JSON mode,
  опциональный `enable_thinking=false` для reasoning-моделей),
  `internal/embed` (embeddings), `internal/rerank` (rerank API) —
  все по OpenAI-compatible протоколу.

`cmd/ml/main.go`: `config.Load()` → `store.New(index.json)` →
клиенты embed/chat/rerank (каждый можно отключить) → `rag.New` →
HTTP-сервер на `:8000`, graceful shutdown по SIGINT/SIGTERM,
`WriteTimeout=300s` (LLM-вызовы долгие).

Отключение компонентов (env-значение `off`/`none` или
`RERANK_ENABLED=false`) переводит сервис в деградационные режимы —
см. ниже.

## Конфигурация

Все настройки — переменные окружения; при старте подхватывается
локальный файл окружения в каталоге сервиса (`KEY=VALUE`, `#` —
комментарии), реальные env-переменные имеют приоритет
(`internal/config/config.go`).

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `PORT` | `8000` | Порт HTTP-сервера |
| `CORS_ORIGIN` | `*` | `Access-Control-Allow-Origin` |
| `LLM_BASE_URL` | `http://127.0.0.1:8081/v1` | Chat-completions endpoint (`off` — без LLM) |
| `LLM_API_KEY` | пусто | Bearer-ключ для LLM endpoint |
| `LLM_MODEL` | `local` | Имя модели |
| `LLM_DISABLE_THINKING` | `true` | `chat_template_kwargs.enable_thinking=false` (Qwen3.5/3-thinking) |
| `LLM_PROMPT_DIR` | `prompts` | Каталог версионируемых промптов |
| `EMBED_BASE_URL` | `http://127.0.0.1:8082/v1` | Embeddings endpoint (`off` — только lexical) |
| `EMBED_API_KEY` / `EMBED_MODEL` | пусто / `local` | Ключ и имя модели эмбеддингов |
| `RERANK_BASE_URL` | `http://127.0.0.1:8083/v1` | Rerank endpoint |
| `RERANK_API_KEY` / `RERANK_MODEL` | пусто / `local` | Ключ и имя реранкера |
| `RERANK_ENABLED` | `true` | Включить cross-encoder rerank |
| `INDEX_PATH` | `index.json` | Путь к JSON-снапшоту индекса |
| `RAG_TOP_K` | `8` | Сколько чанков уходит в генерацию |
| `RAG_MIN_SCORE` | `0.05` | Порог релевантности (score floor) |
| `RETRIEVE_VECTOR_K` | `30` | Глубина векторного кандидатов |
| `RETRIEVE_LEX_K` | `30` | Глубина lexical-кандидатов |
| `RRF_K` | `60` | Константа Reciprocal Rank Fusion |
| `RERANK_CANDIDATES` | `20` | Сколько кандидатов идёт в реранкер |
| `PROMPT_VERSION` | `v1` | Версия промптов (`<name>_<ver>.txt`) |

## HTTP API

`internal/httpapi/server.go`. Тело запросов — JSON до 32 МиБ
(`http.MaxBytesReader`), ошибки — `{"error": "..."}`.

### Публичные (для backend)

| Метод и путь | Назначение |
|---|---|
| `GET /health`, `/health/live` | Liveness + счётчики `{documents, chunks}` |
| `GET /api/documents` | Метаданные проиндексированных документов (без контента) |
| `POST /api/documents` / `PUT /api/documents/{id}` | Upsert документа: чанкинг → эмбеддинги → замена чанков документа → обновление relations |
| `DELETE /api/documents/{id}` | Удалить документ и его чанки (`404` если нет) |
| `POST /api/sync` | `{documents: [...]}` — полная замена индекса (backend зовёт при старте) + перестройка индекса связей |
| `POST /api/ask`, `POST /ask` | `{question, userId?}` → `AskResponse` (legacy `answer`+`sources` + структурные поля) |

### Internal computation API (`/internal/v1/*`)

| Метод и путь | Назначение |
|---|---|
| `POST /documents/parse` | **501** — парсинг выполняет backend-пайплайн, не ML |
| `POST /documents/ocr` | **501** — OCR офлайн (`scripts/parse_corpus.sh`) |
| `POST /embeddings` | `{input: [...]}` (≤256 текстов) → OpenAI-style `{data:[{index, embedding}]}` |
| `POST /rerank` | `{query, documents, top_n?}` → `{results:[{index, relevance_score}]}` |
| `POST /questions/analyze` | Анализ вопроса (intent/domain/subquestions) через `analyze_question` промпт |
| `POST /answers/generate` | `{question}` → структурированный `AnswerPayload` (`AskStructured`) |
| `POST /answers/verify` | `{question, answer}` → вердикт verification-промпта по переиспользованному контексту |
| `POST /relations/extract` | `{title, content}` → LLM-извлечение связей по окнам ~6k символов |
| `GET /relations` | Весь relation-индекс `{docID: [Relation...]}` |
| `GET /stats` | `{documents, chunks, relations}` |

Middleware: CORS (один origin из конфига), recoverer (panic → 500),
логирование запросов. Авторизации нет — сервис рассчитан на
внутренний контур (backend → ML).

## Модель данных (`internal/model`)

- `Document` — id, title, content, tags + регуляторные метаданные
  (`documentNumber`, `documentType`, `documentDate`, `category`,
  `legalStatus`: `in_force|repealed|superseded|draft`). Метаданные
  проталкиваются backend'ом и влияют на предупреждения в ответах.
- `Chunk` — `docID#N`, docTitle, tags, section (пункт), page, text,
  vector. ID вида `<docID>#<index>` — цитаты резолвятся по нему.
- `Source` — цитата: documentId/chunkId, title, section, page,
  documentNumber, legalStatus, excerpt (≤500 рун), score, rerankScore.
- `AnswerPayload` — `status` (`answered|partial|
  clarification_required|not_found|conflict|error`), `summary`,
  `steps[]` (order/title/description/responsible/deadline/condition/
  citation_ids), `warnings`, `missing_information`, `conflicts[]`,
  `sources[]`, `debug` (retrieved/reranked/model/latency для
  офлайн-оценки).
- `AskResponse` — legacy-контракт (`answer`, `sources`) + те же
  структурные поля; `answer` рендерится из payload
  (`renderLegacyAnswer`): summary + нумерованные шаги с маркерами
  `[n]` + блоки «Предупреждения»/«Не хватает информации».

## Конвейер ответа (`internal/rag`)

`AskStructured` (`rag.go`):

```
question
 → retrieveHybrid: векторная нога (embed question → cosine top-K)
   + lexical нога (token overlap top-K) → RRF-merge (k=60)
 → кандидаты обрезаются до RERANK_CANDIDATES (20)
 → rerankChunks: cross-encoder по "Документ/пункт/с./текст" → top-TopK
   (при сбое реранкера — порядок RRF)
 → filterRelevant: score floor RAG_MIN_SCORE, но минимум 3 чанка
   (иначе LLM не сможет честно ответить «не найдено»)
 → пусто на любом этапе → status=not_found + insufficientSummary
 → generateAnswer: промпт generate_answer + контекст [chunk_id] →
   chat JSON mode, temperature 0.1, max_tokens 2500
 → parseAnswerJSON: снять ```json-фенс, вырезать JSON из прозы,
   резолвить citation_ids/алиасы ("1", "[1]", "chunk_…") в реальные
   chunk id, перенумеровать steps с 1
 → verifyPayload: невалидные цитаты отбрасываются; answered без
   цитат → partial + warning; answered без steps/summary → not_found;
   conflicts → status=conflict; источники repealed/superseded → warning
 → sources + debug (полные retrieved/reranked списки для eval)
```

`Ask` — обёртка: `AskStructured` → `renderLegacyAnswer` в текст.

### Ретривал (`retrieve.go` + `store.Search*`)

- **Векторная нога**: `embedder.Embed(question)` → brute-force cosine
  по всем чанкам с векторами (`SearchVector`). При ошибке embed —
  нога пропускается.
- **Lexical нога** (`SearchLexical`): термы запроса — lowercase,
  `ё→е`, длина ≥3, без стоп-слов (~60 русских служебных слов);
  матчинг — точный или общий префикс ≥5 рун (ловит словоформы:
  «конференцию»~«конференции»); словарь `synonyms` (~30 пар
  «бытовое слово → доменная лексика»: «сломался»→«неисправность»);
  бонус +0.15 за совпадение в title/tags; score = overlap·0.85+bonus.
- **RRF**: `score = Σ 1/(k+rank)` по каждой ноге, k=`RRF_K` (60).

### Генерация и верификация

- Промпты — `prompts/<name>_<PROMPT_VERSION>.txt` с фолбэком на
  `<name>.txt` и встроенные дефолты (`rag/defaults.go`); кэш в памяти.
  Файлы: `generate_answer`, `analyze_question`, `verify_answer`,
  `extract_relations` (`ml/prompts/`).
- `buildUserPrompt` формирует контекст: каждый чанк с меткой
  `[chunk_<id>]`, название документа, № и дата приказа, пункт, страница,
  статус `repealed/superseded` (если не `in_force`).
- Жёсткие правила в промпте: только фрагменты, никаких внешних знаний,
  не выдумывать сроки/ответственных/номера — это принцип продукта.
- `AnalyzeQuestion` / `VerifyAnswer` / `ExtractRelationsLLM` —
  отдельные JSON-mode вызовы для internal API (temperature 0);
  markdown-фенс и проза вокруг JSON толерантно вырезаются парсером.

## Индексирование (`internal/store`, `internal/chunk`)

- `Upsert(doc)`: `chunk.Split` → `embed.Embed` батчами → замена чанков
  документа → `saveLocked` (атомарный снапшот). Ошибка embed не роняет
  индексацию — чанки остаются без векторов (только lexical).
- `ReplaceAll(docs)` (`/api/sync`): чанкинг+эмбеддинг всех документов
  до swap'а — читатели никогда не видят полупостроенный индекс;
  после — перестройка relation-индекса.
- `Delete(id)`: документ + чанки + norms; `ErrNotFound` → 404.
- `load()` при старте читает снапшот; `docs` хранит и контент
  (нужен для relations/extract и Doc-обогащения).

### Чанкинг (`internal/chunk/chunk.go`)

- Структурные границы: нумерованные пункты (`3.4.`), заголовки
  («Раздел/Глава/Статья/Приложение/Часть/Порядок/Регламент/Приказ»),
  markdown-заголовки, ALL-CAPS заголовки; мелкие куски <250 рун
  сливаются с соседом.
- Лимиты: `maxChunkRunes=1400`, `overlapRunes=200` (скользящее окно
  для длинных пунктов), page-aware разбиение.
- Маркеры страниц `=== Страница N ===` и `--- page N ---` (эмитит
  OCR-пайплайн) выставляют `Chunk.Page` для цитат; метка в текст
  не попадает.
- `Section` — заголовок пункта, к которому относится чанк.

## Связи документов (`internal/relations`)

Два яруса:

1. **Детерминированный regex** (при индексации/sync): «считать
   утратившим силу приказ № N от DD.MM.YYYY» → `repeals`,
   «изложить/изменить/дополнить … № N от …» → `amends`,
   «утвердить приложение/регламент» → `approves`,
   «в соответствии с / на основании … № N от …» → `references`.
   Evidence — исходный абзац. `ResolveTargets` матчит
   `target_document_number` по корпусу → `target_document_id`.
2. **LLM-ярус** (`/internal/v1/relations/extract`): документ бьётся
   на окна ~6000 символов, JSON-массив связей с
   `target_document_number/date`, `target_title_hint`, `evidence`.

Связи хранятся в памяти `rag.Service.relations` (docID → []Relation),
перестраиваются при `IndexDocument`/`SyncDocuments`/`DeleteDocument`.

## Деградационные режимы

| Отключено | Поведение |
|---|---|
| `EMBED_BASE_URL=off` | Только lexical-нога RRF; векторные кандидаты отсутствуют |
| `LLM_BASE_URL=off` | `degradedPayload`: `status=partial`, summary «модель недоступна», sources — топ-чанки с excerpt ≤600 рун, warning «degraded mode» |
| Ошибка `generateAnswer` | Тот же degraded-fallback (не 500) |
| `RERANK_ENABLED=false` / `RERANK_BASE_URL=off` / ошибка rerank | Порядок RRF без перекорреляции |
| Embed-партия падает при sync | Оставшиеся чанки без векторов, lexical-only |
| Оба LLM и embeddings off | `status=error` + пояснение |
| Пустой/нерелевантный контекст | `status=not_found` + `insufficientSummary` (три возможные причины) |

## Корпус и офлайн-пайплайн

```
ml/data/documents/    — исходники: PDF / DOCX / XLSX (~20 файлов)
ml/scripts/
  parse_corpus.sh     — PDF → pdftotext -layout; <500 символов →
                        pdftoppm 300dpi + tesseract rus+eng;
                        маркеры «=== Страница N ===»
  parse_ooxml.py      — DOCX/XLSX → текст (OOXML, python)
  download_models.sh  — GGUF-веса с HuggingFace в ml/models/
                        (~6 ГБ, в git не хранятся)
ml/data/parsed/       — результат: *.txt на документ (~19 файлов)
ml/data/ocr_pages/    — PNG-страницы для OCR'd PDF (по папкам)
ml/configs/corpus.json— манифест: file → title, documentNumber,
                        documentType, documentDate, category
                        (finance|services|hr|it), legalStatus, tags
```

Манифест сейчас описывает 20 документов, все `in_force`.
`cmd/ingest` backend'а читает этот манифест и грузит документы
в Postgres + пушит в ML (`POST /api/sync` / upsert).

### Модели (внешние, через HTTP)

`download_models.sh` раскладывает GGUF для локального запуска
llama.cpp (ml-сервис их сам не читает — только endpoints):

| Назначение | Порт | Модель |
|---|---|---|
| LLM chat | `:8081` | `Qwen3.5-4B-Q4_K_M.gguf` (+ `Qwen3-4B-Instruct-2507` альтернатива) |
| Embeddings | `:8082` | `Qwen3-Embedding-0.6B-Q8_0.gguf` (`--embedding`) |
| Reranker | `:8083` | `qwen3-reranker-0.6b-q8_0.gguf` (`--reranking`) |

## Команды

```bash
cd ml
go run ./cmd/ml                 # :8000, конфиг из env/локального файла
curl localhost:8000/health
curl -X POST localhost:8000/api/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Как открыть ФЛС?"}'

./scripts/parse_corpus.sh       # переизвлечь тексты корпуса
./scripts/download_models.sh    # скачать GGUF-модели

go build ./... && go test ./... # сборка и тесты
```

## Тесты и оценка

- `internal/chunk/chunk_test.go` — единственный Go-тест (границы
  чанков, page-маркеры, overlap).
- Каталоги `tests/` (unit/integration/evaluation/fixtures),
  `benchmarks/`, `docs/` — заготовки под eval-харнесс; `debug`-блок
  в `AnswerPayload` уже собирает retrieved/reranked трассы для него.

## Ограничения

- In-memory индекс + один JSON-файл: нет конкурентной записи между
  репликами, при росте корпуса нужен pgvector.
- Brute-force cosine по всем векторам — ок для текущего размера.
- Нет авторизации — только внутренний контур.
- `/internal/v1/documents/parse|ocr` — заглушки 501; парсинг/OCR —
  офлайн-скрипты и backend jobs.
- `relations` живут в памяти и не персистятся (перестраиваются из
  корпуса при sync).
- LLM-контекст ограничен релевантными чанками — очень длинные
  регламенты усекаются чанкингом, а не long-context приёмами.
