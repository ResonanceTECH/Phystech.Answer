# API «Физтех.Ответ»

Документация HTTP API backend'а. Каноническая спецификация —
`api/openapi.yaml` (OpenAPI 3.0.3, version 0.2.0); этот файл —
человекочитаемый обзор с приоритетом на фактическую реализацию
(`backend/internal/httpapi/server.go`).

## Базовые URL

| Префикс | Назначение |
|---|---|
| `/api/v1` | Каноническое версионированное API (см. openapi.yaml) |
| `/api` | Не-versioned алиасы для текущего фронтенда |
| `/health` | Liveness/readiness вне `/api` |

Примеры: `http://localhost:8080/api/v1/documents`,
`http://localhost:8080/api/ask`.

ML-сервис имеет собственный API (порт 8000, `/api/documents`,
`/api/ask`, `/internal/v1/*`) — см. `ml/README.md`. Фронтенд ML
напрямую не вызывает.

## Аутентификация и авторизация

- Сессии серверные: при входе backend ставит HttpOnly-cookie
  `fiztech_session` (`SameSite=Lax`, `Secure` — при https
  `APP_PUBLIC_URL`). В БД хранится хеш токена, сырой токен — только
  в cookie. JS cookie не читает — XSS-exfiltration исключена.
- `GET /api/v1/auth/me` возвращает текущего пользователя — фронтенд
  дёргает его один раз при старте для восстановления сессии.
- Роли: `employee` (default при регистрации), `admin`, `expert`.
  `admin` и `expert` имеют административный доступ
  (`requireAdmin` на backend, `isAdminRole` на фронтенде).
- Онбординг: `PATCH /api/v1/users/me` выставляет
  `onboardingComplete`; фронтенд-гард не пускает в `/app` без него.
- Origin-check: mutating-запросы (POST/PUT/PATCH/DELETE) с заголовком
  `Origin` принимаются только от того же host или `CORS_ORIGIN` —
  CSRF-защита для cookie-сессий.
- Rate limiting: auth-эндпоинты ограничены token-bucket по IP
  (in-memory, один инстанс).

### Доступность эндпоинтов

- **Требуют сессию**: `PATCH /users/me`, `saved-instructions`,
  `action-routes*`, весь `/admin/*`.
- **Требуют admin/expert**: все `/api/v1/admin/*`.
- **Открытые**: health, auth-эндпоинты входа/регистрации, каталог
  документов, conversations/messages, runs, search, `/api/ask`,
  список пользователей. `currentUser` опционален — анонимные
  запросы обрабатываются; это осознанное состояние прототипа.

## Формат ошибок

```json
{ "error": { "code": "VALIDATION_ERROR", "message": "...", "retryable": false } }
```

`retryable: true` выставляется для 5xx. Типовые коды:
`VALIDATION_ERROR` (400), `UNAUTHORIZED` (401), `FORBIDDEN` (403),
`NOT_FOUND` (404), `CONFLICT` (409), `EMAIL_ALREADY_EXISTS` (409),
`INVALID_CREDENTIALS` (401), `RATE_LIMITED` (429), `INTERNAL_ERROR` (500).

## Эндпоинты

### System / health

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/health`, `/api/v1/health`, `/health/live` | Liveness, `{status:"ok"}` |
| GET | `/health/ready` | Readiness — проверяет БД |

### Auth и профиль

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/auth/login` | Вход по email → `User` + cookie сессии |
| POST | `/api/v1/auth/register` | Регистрация сотрудника → `User` + cookie |
| POST | `/api/v1/auth/logout` | Отзыв сессии, очистка cookie |
| GET | `/api/v1/auth/me` | Текущий пользователь (или 401/пусто) |
| GET | `/api/v1/users/me` | Алиас `auth/me` |
| PATCH | `/api/v1/users/me` | Обновить профиль: имя, категория, подразделение, должность, `onboardingComplete`. Роль через API не меняется |
| GET | `/api/v1/departments` | Список подразделений для форм |

### Documents (v1)

Базовый объект `Document`: id, title, summary, docType,
documentNumber, version, status (`draft|processing|
requires_review|published|archived`), tags, content, updatedAt.
Чтения возвращают проекцию последней published-версии.

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/documents` | Список документов (query: фильтры) |
| POST | `/api/v1/documents` | Создать документ + version 1 → job parse |
| GET | `/api/v1/documents/{id}` | Карточка документа |
| PUT | `/api/v1/documents/{id}` | Обновить метаданные/контент → новая версия |
| DELETE | `/api/v1/documents/{id}` | Удалить документ |
| POST | `/api/v1/documents/{id}/read` | Отметить прочитанным |
| GET | `/api/v1/documents/{id}/versions` | История версий |
| GET | `/api/v1/documents/{id}/sections` | Разделы документа |
| GET | `/api/v1/documents/{id}/relations` | Связи с другими документами |
| GET | `/api/v1/documents/{id}/attachments` | Вложения |
| GET | `/api/v1/documents/{id}/file` | Исходный файл |
| GET | `/api/v1/documents/{id}/content` | Текст контента |
| GET | `/api/v1/versions/{id}/chunks` | Чанки версии |
| GET | `/api/v1/sources/{id}` | Исходник по source id |

### Search

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/search` | `{query, limit?}` → FTS по published-чанкам (russian-стеммер, OR-термы). Результат: `results[]` с chunk/document/section/page/score |

### Users (v1 + legacy)

| Метод | Путь | Описание |
|---|---|---|
| GET | `/api/v1/users` | Список пользователей |
| POST | `/api/v1/users` | Создать пользователя |
| GET | `/api/users` | Legacy-алиас списка |
| POST | `/api/users` | Legacy-алиас создания |
| GET | `/api/users/{id}/unread` | Legacy: документы, непрочитанные пользователем |

### Conversations и Runs (v1)

Персистентный диалоговый API. `POST .../messages` ставит run в
очередь (`202 {message, run}`); дальше — polling `GET /runs/{id}`
или SSE `GET /runs/{id}/events`; финал — `GET /runs/{id}/answer`.

| Метод | Путь | Описание |
|---|---|---|
| GET/POST | `/api/v1/conversations` | Список / создать диалог `{title?}` → `Conversation` |
| GET/PATCH/DELETE | `/api/v1/conversations/{id}` | Карточка / переименовать / удалить |
| GET/POST | `/api/v1/conversations/{id}/messages` | История / `{text}` → `202 {message, run}` |
| GET | `/api/v1/runs/{id}` | Статус run (`queued|running|done|failed|cancelled`) |
| GET | `/api/v1/runs/{id}/events` | **SSE** `text/event-stream`: `id`/`event`/`data`, события `run.started`, `retrieval.*`, `verification.*`, `answer.completed|failed`, `run.cancelled` |
| GET | `/api/v1/runs/{id}/answer` | `{answer, steps[], citations[]}`; `404` пока не готов |
| POST | `/api/v1/runs/{id}/cancel` | Отмена run |

#### SSE-протокол

- `GET /api/v1/runs/{id}/events` — `text/event-stream`.
- Resume: заголовок `Last-Event-ID` или query `?after=<seq>`.
- События поллятся из `run_events` каждые ~400 мс; heartbeat —
  SSE-комментарии `: ping` каждые ~15 с.
- После терминального статуса (`done|failed|cancelled`) стрим
  дожидается хвост событий и закрывается.

### Answers / Instructions / Action routes

| Метод | Путь | Описание |
|---|---|---|
| POST | `/api/v1/answers/{id}/feedback` | `{rating: positive|negative, comment?}` → 201 |
| GET/POST | `/api/v1/saved-instructions` | Сохранённые инструкции (требуют сессию) |
| GET/POST | `/api/v1/action-routes` | Маршруты действий |
| GET | `/api/v1/action-routes/{id}` | Детали маршрута |
| PATCH | `/api/v1/action-routes/{id}/steps/{stepId}` | Обновить шаг (отметки выполнения и т.п.) |

### Admin (`/api/v1/admin/*`, роль admin/expert)

| Метод | Путь | Описание |
|---|---|---|
| POST | `/admin/documents/upload` | Multipart-загрузка файлов (до 64 МиБ): `.txt/.md/.docx/.xlsx/.pdf` → `source_files` + document + job `document.parse`. Ответ `results[]`: `queued|duplicate|rejected` (дедуп по sha256) |
| GET | `/admin/processing` | Очередь задач `{jobs[]}` |
| GET | `/admin/processing/{id}` | Детали задачи |
| POST | `/admin/processing/{id}/retry` | Повторить задачу |
| POST | `/admin/documents/{id}/publish` | Опубликовать версию |
| POST | `/admin/documents/{id}/reprocess` | Переиндексировать документ |
| POST | `/admin/document-relations` | Создать связь вручную |
| PATCH | `/admin/document-relations/{id}` | Изменить связь |
| POST | `/admin/document-relations/{id}/verify` | Подтвердить/отклонить pending-связь |
| GET | `/admin/dashboard` | Счётчики для админ-панели |

### Legacy `/api/*` (текущий фронтенд)

| Метод | Путь | Описание |
|---|---|---|
| GET/POST/PUT/DELETE | `/api/documents[/{id}]` | CRUD документов (те же хендлеры, что v1) |
| POST | `/api/documents/{id}/read` | Отметка прочтения |
| GET/POST | `/api/users[/{id}/unread]` | Пользователи и непрочитанные |
| POST | `/api/ask` | `{question}` → синхронный ответ `{answer, sources, stub}`; внутри — прокси к `ML /api/ask` + stub-фолбэк при недоступном ML |

`POST /api/ask` — единственный синхронный путь вопроса для UI;
он не создаёт conversation/run записей. Персистентный путь —
`POST /api/v1/conversations/{id}/messages` → run → ML.

## Сквозной поток вопроса (v1)

```
POST /api/v1/conversations/{id}/messages {text}
  → backend: message(user) + answer_run(status=queued) + job answer.generate
  → 202 {message, run}
GET /api/v1/runs/{id}/events (SSE)            — прогресс
GET /api/v1/runs/{id}                         — polling-статус
GET /api/v1/runs/{id}/answer                  — {answer, steps, citations}
```

Воркер (`jobs.answer.generate`): `ML POST /api/ask` → нормализация
статуса → маппинг steps/citations на `document_version_id`/`chunk_id`
→ запись answer → события `answer.completed`/`answer.failed`.
Отмена — `POST /runs/{id}/cancel`; воркер проверяет статус до вызова
ML и перед записью ответа.

## Замечания

- `openapi.yaml` покрывает v1-поверхность; legacy `/api/*` описан
  здесь и в `backend/README.md`.
- Часть полей в спеках — проекции: `Document.version` — строка,
  `status` enum шире backend-овского `processing_status`
  (`pending|processing|published|requires_review|failed`).
- Фронтенд пока использует legacy `/api/documents`, `/api/users`,
  `/api/ask` и v1-auth — versioned conversations/runs/saved-instructions/
  action-routes в UI не задействованы.
- ML `/internal/v1/*` — служебный API для оркестрации вычислений;
  наружу не публикуется.
