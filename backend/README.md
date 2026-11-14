# Backend

Go-бэкенд Phystech.Answer — навигатор сотрудника по непрочитанным регламентам.
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

`POST /api/ask` проксирует запрос на `${ML_SERVICE_URL}/ask` (формат
`{question}` → `{answer, sources}`). Пока ML-сервис не поднят, возвращает
заглушку с `"stub": true`.

## Проверка

```bash
curl localhost:8080/api/health
curl localhost:8080/api/documents
curl -X POST localhost:8080/api/ask -d '{"question":"Когда обед?"}' -H 'Content-Type: application/json'
```
