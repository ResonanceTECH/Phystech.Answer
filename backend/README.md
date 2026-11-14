# Backend

Go-бэкенд «Физтех.Ответ» — REST API, PostgreSQL-хранилище, обработка
документов, фоновая очередь задач и прокси к ML/RAG-сервису.

## Стек

- **Язык**: Go, `net/http` (stdlib `ServeMux` с method+path паттернами).
- **БД**: PostgreSQL через `pgx/v5` (`internal/db`, `internal/store`).
- **Прочие зависимости**: `golang.org/x/crypto`, собственные пакеты
  для парсинга, хранения и очереди задач.

## Структура

```
cmd/
  server/         — основной HTTP-сервер: миграции → сид → ML-синк →
                    встроенный воркер → HTTP
  worker/         — отдельный процесс-воркер (та же очередь задач,
                    для выделения тяжёлой работы)
  ingest/         — разовая загрузка корпуса регламентов из манифеста
internal/
  config/         — конфигурация из переменных окружения
  db/             — пул pgx + раннер встроенных SQL-миграций
  model/          — доменные сущности и JSON-представления
  store/          — интерфейс Store, Postgres-реализация, SeedDemo
  httpapi/        — роутер, middleware, все HTTP-хендлеры
  authx/          — учётные данные и сессионные токены
  mlclient/       — адаптер к ML-сервису (ask, index, sync)
  jobs/           — фоновый воркер очереди задач
  parse/          — извлечение текста из TXT/MD/DOCX/XLSX/PDF,
                    разбиение на разделы и чанки
  storage/        — локальное файловое хранилище исходников
```

## Конфигурация

Все настройки — переменные окружения (шаблон конфигурации лежит
рядом с кодом, реальные env-переменные имеют приоритет над
файловыми). Основные:

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `PORT` / `HTTP_PORT` | `8080` | Порт HTTP-сервера |
| `DATABASE_URL` | — (обязательна) | Строка подключения PostgreSQL |
| `DB_MAX_CONNS` | `10` | Максимум соединений в пуле |
| `CORS_ORIGIN` | `*` | Разрешённые origins через запятую. Для cookie-авторизации `*` не подходит — задайте явный origin фронтенда |
| `ML_SERVICE_URL` | `http://localhost:8000` | Базовый URL ML-сервиса |
| `ML_TIMEOUT_SECONDS` | `120` | Таймаут запросов к ML |
| `SEED_DEMO_DATA` | `true` | Засеять демо-пользователей и документы при старте |
| `STORAGE_DRIVER` | `local` | Драйвер файлового хранилища |
| `STORAGE_PATH` | `./storage` | Каталог загруженных файлов |
| `WORKER_CONCURRENCY` | `2` | Параллельных задач у встроенного воркера |
| `APP_PUBLIC_URL` | `http://localhost:5173` | Публичный URL приложения |
| `SESSION_TTL_HOURS` | `72` | TTL обычной сессии |
| `REMEMBER_TTL_HOURS` | `720` | TTL сессии с «запомнить меня» |
| `ALLOWED_EMAIL_DOMAIN` | пусто — любой домен | Если задан, регистрация только на этот домен почты |

## Запуск

```bash
cd backend
go run ./cmd/server      # http://localhost:8080
```

При старте `cmd/server`:

1. Подключается к PostgreSQL (`db.Connect`, проверка `Ping`).
2. Применяет встроенные миграции `backend/db/migrations/*.sql`
   (номера версий — числовой префикс имени файла, порядок по имени;
   применённые версии — в таблице `schema_migrations`).
3. При `SEED_DEMO_DATA=true` и пустой таблице `users` засевает демо-данные.
4. Создаёт локальное хранилище и ML-клиент.
5. Синхронизирует весь набор документов в ML-индекс (`POST /api/sync`).
6. Запускает фоновый воркер очереди задач в горутине.
7. Поднимает HTTP-сервер с graceful shutdown по SIGINT/SIGTERM.

Отдельный воркер для масштабирования:

```bash
go run ./cmd/worker      # та же очередь, без HTTP
```

## Жизненный цикл документа

```
upload / create  →  document + version 1 (транзакция)
                 →  секции и чанки выводятся из текста
                 →  version.published
                 →  job document.parse (если есть исходный файл)
                 →  parse.Extract → SetVersionContent → published
                 →  ML index upsert
                 →  эвристический детектор связей → document_relations (pending)
```

- Обычные чтения (`GET /documents`) — проекция последней
  published-версии; версии/секции/чанки — источник истины для структуры
  и цитат.
- `processing_status`: `pending | processing | published | requires_review | failed`.
  PDF без текстового слоя и неподдерживаемые форматы уходят в
  `requires_review`.
- `legal_status`: `in_force | repealed | superseded | draft` —
  пробрасывается в ML и влияет на предупреждения в ответах.
- `access_level`: `public | internal | confidential | restricted` —
  `restricted` виден только `admin`/`expert` в поиске.

## Парсинг и структура

`internal/parse`:

- **TXT/Markdown** — как есть.
- **DOCX** — абзацы и таблицы `word/document.xml` (только stdlib zip/xml).
- **XLSX** — листы и строки `ячейка=значение`, объединённые ячейки
  разворачиваются в повторы значения.
- **PDF** — best-effort: распаковка content-stream'ов и сбор
  Tj/TJ-операндов. Скан без текстового слоя → `ErrNoText` →
  `requires_review` (OCR — офлайн-пайплайн в `ml/scripts/`).
- Разбиение на `document_sections`/`document_chunks` — по нумерованным
  пунктам («3.4.») и заголовкам; полная текстовая версия хранится в
  `document_versions.content`.

## Хранилище файлов

`internal/storage` — интерфейс `Storage` + локальная реализация:

- `STORAGE_PATH` — корень; пути нормализуются и проверяются на выход
  за пределы корня (защита от path traversal).
- Ключи вида `files/<sha256><ext>` — одинаковые файлы дедуплицируются
  по контрольной сумме (`idx_source_files_sha`).
- Интерфейс оставлен под замену на S3-совместимое хранилище.

## Очередь задач и воркер

`internal/jobs` — задачи живут в PostgreSQL (`processing_jobs`):

| Тип | Что делает |
|---|---|
| `document.parse` | Извлекает текст из исходника, обновляет версию, пушит документ в ML, детектит связи |
| `answer.generate` | Дёргает ML `/api/ask`, сохраняет answer+steps+citations, пишет события рана |

- Claim — `SELECT … FOR UPDATE SKIP LOCKED` (транзакционная очередь).
- Poll — каждые 500 мс; конкурентность — `WORKER_CONCURRENCY`.
- Зависшие задачи старше 10 минут рекьюятся при старте воркера.
- Ошибки — повтор через 30 с (`retry_at`); `idempotency_key`
  защищает от дублей.
- `answer.generate` эмитит события в `run_events`: `run.started`,
  `retrieval.started`, `retrieval.completed`, `verification.started`,
  `answer.completed`, `answer.failed`, `run.cancelled`.

Отмена рана — `POST /api/v1/runs/{id}/cancel`; воркер проверяет статус
перед вызовом ML и перед записью ответа.

### Автодетект нормативных связей

После парсинга `detectRelations` ищет в тексте паттерны
«отменяется/утратившим силу приказ № N», «внести изменения в приказ № N»
и создаёт `document_relations` со `review_status='pending'`. Подтверждение —
через `/api/v1/admin/document-relations/{id}/verify`. Если документ-цель
не найден в базе, связь не создаётся (принцип «не выдумывать»).

## Аутентификация и роли

- Сессии серверные: в БД — только хеш токена, браузеру — HttpOnly-cookie
  `fiztech_session` (`SameSite=Lax`, `Secure` при https `APP_PUBLIC_URL`).
- Роли: `employee` (по умолчанию при регистрации), `admin`, `expert`.
  Административный доступ — `admin` **и** `expert` (для хакатона
  объединены, см. `requireAdmin`).
- `PATCH /api/v1/users/me` обновляет только профильные поля
  (имя, категория, подразделение, должность, флаг онбординга) —
  роль через API не назначается.
- In-memory token-bucket rate limiting по IP на auth-эндпоинтах
  (один инстанс; при горизонтальном масштабировании — общий стор).
- Origin-check middleware: изменяющие запросы с заголовком `Origin`
  принимаются только от своего host или `CORS_ORIGIN` (CSRF-защита
  для cookie-сессий).

## API

Два поколения эндпоинтов:

- **`/api/v1/*`** — каноническое версионированное API (см.
  `api/openapi.yaml` и `api/README.md`).
- **`/api/*`** — не-versioned алиасы для текущего фронтенда
  (`/api/documents`, `/api/users`, `/api/ask`).

Полные таблицы — в `api/README.md`. Ключевые группы:

| Группа | Пути |
|---|---|
| Health | `GET /api/health`, `/api/v1/health`, `/health/live`, `/health/ready` |
| Auth | `/api/v1/auth/{login,register,logout,me}`, `/api/v1/users/me`, `/api/v1/departments` |
| Documents | `/api/v1/documents*` (+ versions/sections/relations/attachments/file/content), `/api/v1/versions/{id}/chunks`, `/api/v1/sources/{id}` |
| Search | `POST /api/v1/search` — FTS по опубликованным чанкам |
| Users | `/api/v1/users`, `/api/users`, `/api/users/{id}/unread` |
| Conversations | `/api/v1/conversations*`, `.../messages` |
| Runs | `/api/v1/runs/{id}`, `.../events` (SSE), `.../answer`, `.../cancel` |
| Feedback и прочее | `/api/v1/answers/{id}/feedback`, `/api/v1/saved-instructions`, `/api/v1/action-routes*` |
| Admin | `/api/v1/admin/*` — upload, processing, publish/reprocess, relations, dashboard |
| Legacy ask | `POST /api/ask` — синхронный прокси к ML (плоский ответ + stub-фолбэк) |

### Формат ошибок

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "retryable": false } }
```

`retryable` выставляется для 5xx. Коды: `VALIDATION_ERROR`,
`UNAUTHORIZED`, `FORBIDDEN`, `NOT_FOUND`, `CONFLICT`,
`EMAIL_ALREADY_EXISTS`, `INVALID_CREDENTIALS`, `RATE_LIMITED`,
`INTERNAL_ERROR` и др.

### Доступность эндпоинтов

- **Требуют сессию**: `PATCH /users/me`, `saved-instructions`,
  `action-routes*`, весь `/admin/*` (плюс `admin`/`expert` для admin).
- **Открытые**: health, auth-эндпоинты, каталог документов,
  conversations/messages, runs, search, `/api/ask`, список пользователей.
  `currentUser` используется опционально — анонимные запросы
  обрабатываются с пустым пользователем. Это осознанное состояние
  прототипа; для продакшена каталог и диалоги стоит закрыть сессией.

## Поиск

`POST /api/v1/search` — PostgreSQL FTS:

- `to_tsvector('russian', lower(content))` — русский стеммер;
  `lower()` обязателен, т.к. стеммер не нормализует слова с заглавной.
- GIN-индекс `idx_chunks_fts` (миграция `0002_fts_lower`).
- Запрос — термы пользователя, санитизированные от синтаксиса tsquery,
  соединяются по `OR` (`string_agg(t, ' | ')` → forgiving recall).
- Ищет только по чанкам `published`-версий; `restricted` документы
  отфильтровываются для не-привилегированных ролей.
- Семантический поиск живёт в ML-сервисе — backend лишь проксирует `/ask`.

## Интеграция с ML

`internal/mlclient`:

- `Ask` → `POST {ML_SERVICE_URL}/api/ask` — ждёт и плоский
  `{answer, sources}`, и структурный payload (`status`, `steps`,
  `warnings`, `missing_information`, `conflicts`).
- `IndexDocument`/`RemoveDocument` → `PUT`/`DELETE /api/documents/{id}`.
- `SyncDocuments` → `POST /api/sync` — полная замена индекса при старте.
- Недоступный ML не ломает API: `/api/ask` и `answer.generate` отдают
  stub-ответ с `"stub": true` / предупреждением degraded.

## Фоновая обработка ответов

`answer.generate` превращает ML-ответ в персистентную модель:

- `status` — из ML, нормализуется к закрытому набору
  (`answered|partial|clarification_required|not_found|conflict|error`).
- Если ML вернул `steps` — шаги и привязка цитат по `citation_ids`
  (chunkID → порядковый номер источника). Иначе — legacy-разбор
  нумерованных строк ответа и маркеров `[n]`.
- Цитаты резолвятся в `document_version_id`/`chunk_id` по
  совпадению `section_number` (best effort, промахи остаются NULL).
- `res.Stub` и `conflicts` добавляются в `warnings`.

## Загрузка файлов (admin)

`POST /api/v1/admin/documents/upload` — multipart (`files`/`file`),
до 64 MiB на запрос:

- Поддерживаемые расширения — по `parse.Supported`
  (.txt/.md/.docx/.xlsx/.pdf).
- SHA-256 дедупликация → `status: "duplicate"` без повторной обработки.
- Файл → `source_files` + content-addressed ключ в storage;
  документ + version 1 → `pending` → job `document.parse`.
- Поля формы `title`, `tags` (CSV), `category`, `documentType`,
  `documentNumber` прокидываются в метаданные документа.
- Ответ — массив `results[]` со статусами `queued|duplicate|rejected`.

## Сид-данные

`store.SeedDemo` (только при пустой `users`): демо-пользователи
(employee/admin/expert, онбординг пройден) и несколько синтетических
регламентов (командировки, ФЛС, заимствование, кадры, рабочее время),
плюс одна подтверждённая связь `repeals`. Сид пропускается, если база
уже заполнена.

## Команды

```bash
go run ./cmd/server     # сервер + воркер
go run ./cmd/worker     # только воркер
go run ./cmd/ingest -manifest ../ml/configs/corpus.json \
    -parsed ../ml/data/parsed -ml http://localhost:8000
go build ./...          # сборка
go test ./...           # юнит-тесты
```

Проверка:

```bash
curl localhost:8080/api/health
curl localhost:8080/health/ready
curl -X POST localhost:8080/api/ask \
  -H 'Content-Type: application/json' \
  -d '{"question":"Как открыть ФЛС?"}'
```

## Замечания и ограничения

- Встроенный воркер и отдельный `cmd/worker` могут работать
  одновременно — очередь разруливает `SKIP LOCKED`; при разделении
  следите, чтобы не было двух HTTP-серверов на один порт.
- Rate limiting — per-process; несколько реплик backend требуют
  внешнего стора.
- `GET /admin/processing/{id}` ищет задачу полным сканом
  `ListJobs(limit=500)` — при большой очереди заменить на точечный запрос.
- `chunk_embeddings` в схеме предусмотрена под pgvector, но сейчас
  эмбеддинги живут только в ML-сервисе.
- Тестовое покрытие точечное: `authx`, `ratelimit`.
