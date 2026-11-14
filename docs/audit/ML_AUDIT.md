# ML_AUDIT — Физтех.Ответ

Go-сервис `ml/`: индекс в `index.json`+store, RAG в `internal/rag` (retrieve/rerank/chat/relations), chunk/embed/rerank модули. Модели — **локальные llama.cpp (GGUF) на хосте**: `LLM_BASE_URL=host.docker.internal:8081/v1`, `EMBED_BASE_URL=:8082`, `RERANK_BASE_URL=:8083` (compose). Внешних API нет → конфиденциальность соблюдена. ✅

## Компоненты

| Компонент | Реализация | Статус |
|---|---|---|
| Chunking | `internal/chunk` (+test) | ✅ |
| Embeddings | `internal/embed` → `/v1/embeddings` локальная модель | ✅ real |
| Retrieve | `rag/retrieve.go` hybrid FTS+vector | ✅ |
| Rerank | `internal/rerank` | ✅ real |
| LLM chat | `internal/chat` → локальный GGUF | ✅ real |
| Prompts | `internal/prompts` + `ml/prompts/*.txt` versioned | ✅ |
| Verify | `VerifyAnswerWithContext` | ✅ |
| Relations | `internal/relations` + `cmd/relcheck` utility | ✅ |
| Parse/OCR endpoints | `/internal/v1/documents/{parse,ocr}` | ❌ 501 (backend pipeline) |

## Проверено вживую

- `/health`: 28 docs / 1568 chunks.
- `/api/ask` «вернуть деньги за проезд» → status=answered, 8 sources (включая свежезагруженный приказ), steps с [n]-цитатами.
- Persistent run «отпуск за свой счёт» → status=partial, честные missing_information.
- **No-answer**: «вид на жительство» → status=not_found, без галлюцинаций, missing заполнен. ✅
- **Актуальность**: корпус содержит `Приказ № 5990-1 от 03.11.2026` с confirmed `repeals`→«Порядок внутреннего заимствования (ред. 2024)»; worker-детектор кладёт кандидатов в pending на проверку экспертом.
- **Деградация**: ML down → stub-ответ `Stub:true`+warning; index/remove ошибки только логируются; `/api/sync` ре-синк на старте backend.

## Не проверено / ограничения

- **OCR**: runtime OCR отсутствует (501); скан-PDF сценарий → `requires_review`. Тест со сканом не проводился — OCR только offline (`scripts/parse_corpus.sh`).
- **Метрики качества** (Recall@5/MRR/nDCG/Citation Accuracy/Unsupported Claim Rate): эталонного размеченного набора нет — не измерялись. Нужен ручной eval для защиты.
- Содержательная проверка «цитата подтверждает смысл» — verify-stage есть, ручная выборка не проводилась.
- PDF/DOCX/XLSX parse E2E не гонялся (код `internal/parse` есть; .txt проверен).
- Uncommitted WIP: `rag.go` citation aliases `chunk_<id>` (коммит 2d93f98 частично покрыл).
