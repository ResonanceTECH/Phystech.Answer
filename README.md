# Phystech.Answer — Физтех.Ответ

Интеллектуальный навигатор по внутренним регламентам и административным
процессам МФТИ. Сотрудник задаёт вопрос обычным языком — система находит
подходящие документы, формирует пошаговую инструкцию и показывает
источники (документ + пункт + цитата).

Ключевой принцип: **ответ только по базе знаний**. Сервис не придумывает
сроки, ответственных, номера приказов и требования — если подтверждений
во фрагментах нет, ответ честно помечается `partial`,
`not_found` или `conflict`, а не «додумывается».

## Архитектура

```
┌──────────┐   HTTP/JSON    ┌──────────┐   HTTP/JSON   ┌───────────────┐
│ frontend │ ─────────────► │ backend  │ ────────────► │ ml (RAG)      │
│ React 19 │  cookie-сессия │ Go :8080 │               │ Go :8000      │
│ Vite+MUI │                │          │               │               │
└──────────┘                └────┬─────┘               └───────┬───────┘
                                 │                             │
                          ┌──────▼──────┐              OpenAI-compatible
                          │ PostgreSQL  │              HTTP API
                          │ 17 :5432    │              ┌───────▼───────┐
                          └─────────────┘              │ LLM / embed / │
                                                       │ rerank модели │
                                                       │ (llama.cpp)   │
                                                       └───────────────┘
```

| Сервис | Стек | Назначение |
|---|---|---|
| `frontend/` | React 19, TypeScript, Vite, MUI, TanStack Query, Zustand, FSD | SPA: лендинг, чат-ассистент, каталог документов, профиль, админ-раздел |
| `backend/` | Go 1.25, `net/http`, pgx/v5 | REST API `/api/v1` + legacy `/api`, PostgreSQL, очередь задач, синхронизация документов в ML |
| `ml/` | Go 1.25, чистый stdlib | RAG: чанкинг, гибридный поиск (vector+lexical+RRF), rerank, структурированная генерация и верификация ответа |
| `postgres` | PostgreSQL 17 | Документы, версии, чанки, пользователи, сессии, задачи, раны |
| `mailpit` | axllent/mailpit | Перехватчик исходящей почты для разработки (веб :8025) |

Внешние зависимости — только inference-серверы с OpenAI-compatible API
(llama.cpp/Ollama): chat-модель, эмбеддинги, реранкер. Без них сервис
работает в деградированных режимах (см. ниже).

### Потоки запросов

**Синхронный (текущий UI):**
`POST /api/ask {question}` → backend проксирует в `ml POST /api/ask` →
`{answer, sources, stub}`. ML недоступен → stub-фолбэк.

**Персистентный (API v1, UI пока не использует):**
`POST /api/v1/conversations/{id}/messages {text}` → `202 {message, run}`
→ фоновая задача `answer.generate` → ML `/api/ask` → run-events
(SSE `/api/v1/runs/{id}/events`) → `GET /api/v1/runs/{id}/answer`
`{answer, steps[], citations[]}`.

**Индексация:**
backend при старте и после CRUD документов пушит их в ML
(`PUT/DELETE /api/documents/{id}`, при старте — `POST /api/sync`);
ML чанкует текст, считает эмбеддинги и кладёт в индекс.

## Документация

| Документ | Содержание |
|---|---|
| [backend/README.md](backend/README.md) | Архитектура Go-бэкенда, пакеты, PostgreSQL и миграции, очередь задач, auth/роли, API, поиск, интеграция с ML, хранение файлов |
| [frontend/README.md](frontend/README.md) | React-приложение: FSD-структура, маршруты и guard'ы, стейт-менеджмент, сценарии чата/документов/админки, тесты, ограничения |
| [ml/README.md](ml/README.md) | RAG-конвейер: retrieval (vector+lexical+RRF), rerank, генерация и верификация, промпты, relations, degraded-режимы, корпус и модели |
| [api/README.md](api/README.md) | Полный каталог HTTP API: auth-модель, эндпоинты по группам, SSE-протокол runs, форматы ошибок |
| [api/openapi.yaml](api/openapi.yaml) | OpenAPI 3.0.3 спецификация versioned API (`/api/v1`) |

## Быстрый старт

Требуется Docker с плагином Compose:

```bash
docker compose up --build        # http://localhost:3000
```

| Сервис | Порт снаружи | Назначение |
|---|---|---|
| frontend | **3000** | Веб-приложение (nginx: статика + прокси `/api` → backend) |
| backend | 8080 | REST API |
| ml | 8000 | RAG-сервис |
| postgres | 5432 | PostgreSQL (проброшен для локальных утилит) |
| mailpit | 8025 / 1025 | Перехватчик исходящей почты для разработки |

При старте стек поднимается сам:

1. `postgres` — healthcheck `pg_isready`.
2. `ml` — загружает `index.json` (пустой при первом запуске), ждёт
   healthcheck.
3. `backend` — после healthy postgres+ml: применяет миграции
   `backend/db/migrations/`, при `SEED_DEMO_DATA=true` засевает
   демо-данные (если БД пуста), создаёт файловое хранилище,
   синхронизирует все документы в ML (`POST /api/sync`), запускает
   встроенный воркер очереди задач, поднимает HTTP.
4. `frontend` — статика + nginx-прокси `/api` → `backend:8080`
   (таймауты до 300 с под долгие LLM-вызовы, SPA-fallback на
   `index.html`).

Руками ничего настраивать не нужно — все значения имеют дефолты.

### Проверка

```bash
curl localhost:8080/api/health     # {"status":"ok"}
curl localhost:8080/health/ready   # readiness (проверяет БД)
curl localhost:8000/health         # {"status":"ok","documents":N,"chunks":M}
curl -X POST localhost:8080/api/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Как открыть финансово-лицевой счёт?"}'
```

## Данные и тома

| Volume | Что хранит |
|---|---|
| `pgdata` | PostgreSQL: документы, версии, чанки, пользователи, сессии, задачи, раны |
| `ml-data` | ML-индекс (`/data/index.json`) |
| `backend-data` | Загруженные исходные файлы регламентов (`/data/files`) |

Данные переживают `docker compose down` и `restart`. Полный сброс —
`docker compose down -v`.

## Конфигурация

`docker compose` подставляет переменные окружения в сервисы; все
значения имеют дефолты (compose-файл — `docker-compose.yml`,
детальные таблицы — в README каждого сервиса).

| Переменная | Дефолт | Назначение |
|---|---|---|
| `SEED_DEMO_DATA` | `true` | Засеять демо-пользователей и документы при старте backend |
| `CORS_ORIGIN` | `http://localhost:3000` | Разрешённые origins через запятую. Cookie-сессии не работают с `*` — задайте origin фронта |
| `APP_PUBLIC_URL` | `http://localhost:3000` | Публичный URL приложения; `https://` включает Secure-cookie |
| `SESSION_TTL_HOURS` / `REMEMBER_TTL_HOURS` | `72` / `720` | TTL сессии без/с «запомнить меня» |
| `ALLOWED_EMAIL_DOMAIN` | пусто | Если задан — регистрация только на этот домен почты |
| `LLM_BASE_URL` | `http://host.docker.internal:8081/v1` | Chat-completions endpoint (`off` — без LLM) |
| `LLM_API_KEY` / `LLM_MODEL` | пусто / `local` | Ключ и имя chat-модели |
| `LLM_DISABLE_THINKING` | `true` | `enable_thinking=false` для reasoning-моделей |
| `EMBED_BASE_URL` / `EMBED_API_KEY` / `EMBED_MODEL` | `…:8082/v1` / пусто / `local` | Embeddings endpoint |
| `RERANK_BASE_URL` / `RERANK_API_KEY` / `RERANK_MODEL` / `RERANK_ENABLED` | `…:8083/v1` / пусто / `local` / `true` | Cross-encoder reranker |
| `RAG_TOP_K` / `RAG_MIN_SCORE` | `8` / `0.05` | Глубина контекста и порог релевантности |
| `VITE_APP_NAME` / `VITE_DEMO_MODE` / `VITE_ENABLE_ADMIN` | `Физтех.Ответ` / `true` / `true` | Build-args фронтенда |

Backend дополнительно читает `DATABASE_URL` (строка подключения к
postgres — в compose собирается из настроек сервиса `postgres`),
`STORAGE_PATH`, `WORKER_CONCURRENCY`, `ML_SERVICE_URL`,
`ML_TIMEOUT_SECONDS`, `DB_MAX_CONNS` — см. `backend/README.md`.

## Inference: локальные модели

По умолчанию compose ожидает три OpenAI-compatible сервера на
хост-машине (`host.docker.internal` работает в Docker Desktop и
проброшен через `extra_hosts` на Linux):

```bash
# chat-модель на :8081, эмбеддинги на :8082, реранкер на :8083
llama-server -m ml/models/llm/Qwen3.5-4B-Q4_K_M.gguf --port 8081
llama-server -m ml/models/embedding/Qwen3-Embedding-0.6B-Q8_0.gguf \
    --port 8082 --embedding
llama-server -m ml/models/reranker/qwen3-reranker-0.6b-q8_0.gguf \
    --port 8083 --reranking
```

GGUF-веса (~6 ГБ) в git не хранятся — скачиваются скриптом:

```bash
ml/scripts/download_models.sh              # весь набор
ml/scripts/download_models.sh qwen3.5      # фильтр по имени
```

| Файл (`ml/models/`) | Назначение |
|---|---|
| `llm/Qwen3.5-4B-Q4_K_M.gguf` | Chat/reasoning (~2.8 ГБ); альтернатива `llm/Qwen3-4B-Instruct-2507-Q4_K_M.gguf` (~1.8 ГБ) |
| `embedding/Qwen3-Embedding-0.6B-Q8_0.gguf` | Эмбеддинги (~0.6 ГБ) |
| `reranker/qwen3-reranker-0.6b-q8_0.gguf` | Реранкер (~0.6 ГБ) |

Альтернатива — любой OpenAI-compatible API: пропишите `LLM_BASE_URL`,
`LLM_API_KEY`, `LLM_MODEL` (и `EMBED_*`, `RERANK_*`) в переменных
окружения compose.

### Деградация без моделей

| Что недоступно | Поведение |
|---|---|
| Embeddings (`EMBED_BASE_URL=off`) | Только lexical-поиск; векторная нога отключается |
| LLM (`LLM_BASE_URL=off` или ошибка генерации) | `status=partial`, ответ — релевантные фрагменты с excerpt'ами, warning «degraded mode» |
| Реранкер | Порядок RRF без перекорреляции |
| ML целиком | backend `/api/ask` отдаёт stub-ответ (`"stub": true`) |

Для демо без GPU допустимо поднять стек с `LLM_BASE_URL=off
EMBED_BASE_URL=off` — поиск и цитаты работают, генерации нет.

## Локальная разработка

Отдельные сервисы без compose (нужен запущенный PostgreSQL — роль и
база создаются `backend/db/init.sql`):

```bash
# ML-сервис → :8000 (настройки — переменные окружения, см. ml/README.md)
cd ml && go run ./cmd/ml

# Backend → :8080 (DATABASE_URL и пр. — см. backend/README.md)
cd backend && go run ./cmd/server

# Frontend → :5173 (прокси /api → :8080)
cd frontend && npm install && npm run dev
```

Проверки: `go build ./...` / `go test ./...` в `backend` и `ml`;
`npm run build|lint|test` во `frontend`.

## Обслуживание

```bash
docker compose ps                  # статус и health
docker compose logs -f ml          # логи сервиса
docker compose restart backend     # перезапуск одного сервиса
docker compose up -d --build       # пересборка и подъём в фоне
docker compose exec postgres \
  psql -U fiztech_admin -d fiztech_answer   # psql в контейнере
```

Backend при старте применяет миграции автоматически — ручных шагов
для обновления схемы нет.

## Структура репозитория

```
├── api/           — openapi.yaml + README (каталог API)
├── backend/       — Go API, миграции, jobs, парсеры, хранилище
│   ├── cmd/       — server | worker | ingest
│   ├── db/        — init.sql + migrations
│   └── internal/  — config, db, model, store, httpapi, authx,
│                    mlclient, jobs, parse, storage
├── frontend/      — React SPA (app/pages/widgets/features/entities/shared)
├── ml/            — Go RAG-сервис
│   ├── cmd/ml/    — точка входа
│   ├── internal/  — rag, chunk, embed, rerank, chat, store,
│   │                relations, prompts, httpapi, config, model
│   ├── prompts/   — версионируемые промпты (analyze/generate/verify/extract)
│   ├── configs/   — corpus.json (манифест нормативной базы)
│   ├── data/      — documents (исходники), parsed (тексты), ocr_pages
│   ├── scripts/   — parse_corpus.sh, parse_ooxml.py, download_models.sh
│   └── models/    — GGUF-веса (не в git)
└── docker-compose.yml
```

## Статус и ограничения

- Чат UI работает на синхронном `/api/ask`; персистентные
  conversations/runs/SSE на backend реализованы, но не подключены
  к фронтенду.
- Текущий чат рендерит плоский ответ; структурированные
  `steps`/`citations` из ML-payload бэкенд сохраняет, но UI их
  пока не показывает.
- Диалоги и сохранённые инструкции на фронтенде — localStorage,
  без серверной синхронизации.
- Админ-загрузка файлов в UI принимает только текстовые форматы;
  backend-парсинг PDF/DOCX/XLSX и admin upload API существуют,
  но UI их не использует. Четыре раздела админки — заглушки.
- ML-индекс — in-memory + JSON-снапшот; при росте корпуса нужен
  pgvector. Парсинг/OCR — офлайн-пайплайн, runtime-эндпоинты
  `/internal/v1/documents/{parse,ocr}` возвращают 501.
- Rate limiting backend — per-process; несколько реплик требуют
  внешнего стора.
