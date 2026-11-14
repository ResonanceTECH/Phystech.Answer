# Phystech.Answer — Физтех.Ответ

Интеллектуальный навигатор по внутренним регламентам и административным
процессам МФТИ. Сотрудник задаёт вопрос обычным языком — система находит
подходящие документы, формирует пошаговую инструкцию и показывает
источники (документ + пункт).

## Архитектура

```
frontend/   React 19 + TypeScript + Vite + MUI — FSD-структура
            (app/pages/widgets/features/entities/shared), чат-ассистент,
            каталог документов, админ-раздел
backend/    Go (stdlib) — REST API документов/сотрудников, прокси /api/ask,
            синхронизация индекса с ML-сервисом
ml/         Go (stdlib) — RAG: чанкинг, эмбеддинги, векторный+лексический
            поиск, генерация ответа через OpenAI-compatible LLM API
```

Цепочка запроса: `POST /api/ask` (backend :8080) → `POST /api/ask` (ml :8000)
→ retrieval по чанкам регламентов → LLM → `{answer, sources[]}`.

## Запуск

```bash
# 1. ML-сервис (RAG)
cd ml && cp .env.example .env && go run ./cmd/ml     # :8000

# 2. Backend API
cd backend && cp .env.example .env && go run ./cmd/server   # :8080

# 3. Frontend
cd frontend && npm install && npm run dev            # :5173
```

Документы из backend автоматически синхронизируются в ML-индекс при старте
и при CRUD. LLM и эмбеддинги настраиваются через `ml/.env`
(OpenAI-compatible API; по умолчанию — локальный Ollama). Без них сервис
работает в деградированном режиме: лексический поиск + ответы-экстракты.

Детали — в README соответствующих директорий (`backend/`, `ml/`, `frontend/`).
