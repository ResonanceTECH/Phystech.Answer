# Backend

Go-бэкенд Phystech.Answer — API документов и сотрудников, прокси вопросов
к ML-сервису (RAG-ассистент), синхронизация индекса.
Только стандартная библиотека, без внешних зависимостей.

## Запуск

```bash
cd backend
go run ./cmd/server        # http://localhost:8080
```

Конфигурация — через переменные окружения (подхватывается `.env`, см. `.env.example`).

## Структура

```
cmd/server/         — точка входа
internal/config/    — конфиг из env + .env
internal/model/     — сущности (Document, User)
internal/store/     — интерфейс хранилища + in-memory реализация + сид
internal/httpapi/   — роутер, middleware (CORS, лог, recover), хендлеры
```

Хранилище сейчас в памяти; чтобы перейти на БД, реализуйте `store.Store`
(Postgres/SQLite) и подмените `store.NewMemory()` в `cmd/server/main.go`.

## API

| Method | Path                        | Описание                                    |
|--------|-----------------------------|---------------------------------------------|
| GET    | /api/health                 | health-check                                |
| GET    | /api/documents?userId=      | список регламентов (+ статус прочтения)     |
| POST   | /api/documents              | создать регламент `{title, content, tags}`  |
| GET    | /api/documents/{id}         | получить регламент                          |
| PUT    | /api/documents/{id}         | обновить регламент (version++)              |
| DELETE | /api/documents/{id}         | удалить регламент                           |
| POST   | /api/documents/{id}/read    | отметить прочитанным `{userId}`             |
| GET    | /api/users                  | список сотрудников                          |
| POST   | /api/users                  | создать сотрудника `{name, email}`          |
| GET    | /api/users/{id}/unread      | непрочитанные регламенты сотрудника         |
| POST   | /api/ask                    | вопрос к регламентам `{question}` → прокси к ML |

`POST /api/ask` проксирует запрос на `${ML_SERVICE_URL}/api/ask` и возвращает
`{answer, sources: [{documentId, title, section, excerpt, score}], stub}`.
При недоступном ML-сервисе возвращает заглушку с `"stub": true`.

## Синхронизация с ML-сервисом

При старте backend отправляет все документы в ML (`POST /api/sync`),
а при создании/обновлении/удалении документа — точечные `PUT`/`DELETE`
`/api/documents/{id}`. Ошибки синка логируются, но не ломают API.

## Проверка

```bash
curl localhost:8080/api/health
curl localhost:8080/api/documents
curl -X POST localhost:8080/api/ask -d '{"question":"Как открыть ФЛС?"}' -H 'Content-Type: application/json'
```
