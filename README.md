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

## Развёртывание

### Docker Compose — весь стек одной командой

Требуется Docker с плагином Compose. Из корня репозитория:

```bash
docker compose up --build        # http://localhost:3000
```

Поднимаются пять сервисов, зависимости и health-check'и настроены
автоматически:

| Сервис    | Что это                          | Порт снаружи | Порт внутри |
|-----------|----------------------------------|--------------|-------------|
| frontend  | React-сборка + nginx, SPA        | **3000**     | 80          |
| backend   | Go API, миграции, сид, воркеры   | 8080         | 8080        |
| ml        | Go RAG-сервис                    | 8000         | 8000        |
| postgres  | PostgreSQL 17                    | — (только внутри сети) | 5432 |
| mailpit   | Перехватчик SMTP (письма восстановления) | **8025** (web), 1025 (smtp) | 8025/1025 |

Цепочка: браузер → nginx (frontend :3000) → `/api` проксируется на
backend → backend ходит в postgres и в ml → ml дёргает LLM/эмбеддинги.
Письма восстановления пароля backend отправляет в mailpit —
прочитать их можно в веб-интерфейсе http://localhost:8025.

При старте backend сам применяет миграции (`db/migrations/`), засевает
демо-данные (`SEED_DEMO_DATA=true`) и синхронизирует документы в
ML-индекс — руками ничего делать не нужно.

## Авторизация

Полный цикл аккаунта реализован и работает поверх PostgreSQL:

- **Маршруты фронта**: `/login`, `/register`, `/forgot-password`,
  `/reset-password`, `/onboarding`, `/app/profile` (плюс `/app/*`, `/admin`).
- **Сессии** — серверные, в БД хранится только SHA-256 хеш токена;
  браузер получает HttpOnly+SameSite=Lax cookie (`Secure` при https).
  «Запомнить меня» продлевает TTL сессии.
- **Пароли** — Argon2id для новых аккаунтов; bcrypt-хеши ранних
  пользователей продолжают проверяться.
- **Восстановление** — одноразовые токены (20 мин по умолчанию),
  ссылка ведёт на `APP_PUBLIC_URL/reset-password?token=…`; при сбросе
  все сессии пользователя отзываются. Письма уходят через SMTP
  (в compose — Mailpit, http://localhost:8025).
- **Onboarding** — после регистрации пользователь уходит на `/onboarding`
  (категория, подразделение, должность); до завершения защищённые
  маршруты редиректят туда.
- **Rate limiting** — на login/register/forgot/reset по IP.
- **Роли** — `employee`, `admin`, `expert`; админ-раздел `/admin`
  доступен `admin` и `expert` (проверка на backend).

Демо-аккаунты (сеются при `SEED_DEMO_DATA=true`):
`ivan.petrov@example.com / ivan123`, `maria.sidorova@example.com / maria123`,
`admin@example.com / admin123`, `expert@example.com / expert123`.
Также работает вход по старому логину (`ivan`, `admin`, …).

#### Персистентные данные (docker volumes)

| Volume         | Что хранит                              |
|----------------|-----------------------------------------|
| `pgdata`       | База PostgreSQL (документы, пользователи, ответы) |
| `ml-data`      | ML-индекс (`index.json`)                |
| `backend-data` | Загруженные исходные файлы регламентов  |

Данные переживают `docker compose down` и `restart`. Полный сброс —
`docker compose down -v`.

#### Конфигурация — `.env` в корне

`docker compose` читает `.env` из корня репо (см. `.env.example`,
все значения имеют дефолты — можно вообще без файла). Ключевые:

| Переменная        | Дефолт                              | Назначение |
|-------------------|-------------------------------------|------------|
| `POSTGRES_PASSWORD` | `fiztech_dev_hack`                | Пароль БД; попадает и в `DATABASE_URL` backend'а. Спецсимволы — только URL-encoded |
| `SEED_DEMO_DATA`  | `true`                              | Сеять демо-данные при старте backend |
| `LLM_BASE_URL`    | `http://host.docker.internal:11434/v1` | OpenAI-compatible endpoint для генерации ответов |
| `LLM_API_KEY`     | пусто                               | Ключ LLM API |
| `LLM_MODEL`       | `qwen2.5:7b`                        | Модель чата |
| `EMBED_BASE_URL`  | `http://host.docker.internal:11434/v1` | Endpoint эмбеддингов |
| `EMBED_MODEL`     | `nomic-embed-text`                  | Модель эмбеддингов |
| `RAG_TOP_K`       | `8`                                 | Сколько чанков идёт в контекст |
| `CORS_ORIGIN`     | `http://localhost:3000`             | Разрешённые origins через запятую (cookie-сессии не работают с `*`) |
| `APP_PUBLIC_URL`  | `http://localhost:3000`             | Публичный URL фронта — база ссылок в письмах; `https://` включает Secure-cookie |
| `SMTP_HOST`/`SMTP_PORT` | `mailpit`/`1025`              | SMTP для писем восстановления пароля (в compose — mailpit) |
| `SMTP_USERNAME`/`SMTP_PASSWORD`/`SMTP_FROM` | пусто / `1025` / `Физтех.Ответ <no-reply@…>` | SMTP-авторизация и адрес отправителя |
| `SESSION_TTL_HOURS` / `REMEMBER_TTL_HOURS` | `72` / `720` | TTL сессии без/с «Запомнить меня» |
| `PASSWORD_RESET_TTL_MINUTES` | `20`                    | Срок жизни ссылки восстановления пароля |
| `ALLOWED_EMAIL_DOMAIN` | пусто                        | Если задан — регистрация только на этот домен почты |
| `VITE_APP_NAME` / `VITE_DEMO_MODE` / `VITE_ENABLE_ADMIN` | `Физтех.Ответ` / `true` / `true` | Build-args фронта |

#### LLM: полный режим vs деградированный

По умолчанию ml ожидает **Ollama на хост-машине** (`host.docker.internal`
работает в Docker Desktop на macOS/Windows и проброшен через
`extra_hosts` на Linux):

```bash
ollama pull qwen2.5:7b        # чат
ollama pull nomic-embed-text  # эмбеддинги
ollama serve                  # если ещё не запущен
```

Тогда ответы будут полноценными, со сгенерированной инструкцией.

Если LLM недоступна (`LLM_BASE_URL=off` или Ollama не поднят) — сервис
работает в деградированном режиме: лексический поиск + ответы-экстракты
из найденных фрагментов. Это нормально для демо без GPU.

Альтернатива — любой OpenAI-compatible API: пропишите `LLM_BASE_URL`,
`LLM_API_KEY`, `LLM_MODEL` (и аналогично `EMBED_*`) в корневой `.env`.

#### Обслуживание

```bash
docker compose logs -f ml         # логи конкретного сервиса
docker compose restart backend    # перезапуск одного сервиса
docker compose up -d --build      # пересобрать и поднять в фоне
docker compose ps                 # статус и health
docker compose exec postgres psql -U fiztech_admin -d fiztech_answer  # psql
```

### Локально (для разработки)

```bash
# 0. PostgreSQL (нужен backend'у, см. DATABASE_URL в backend/.env;
#    инициализация роли/базы — backend/db/init.sql)

# 1. ML-сервис (RAG)
cd ml && cp .env.example .env && go run ./cmd/ml     # :8000

# 2. Backend API
cd backend && cp .env.example .env && go run ./cmd/server   # :8080

# 3. Frontend
cd frontend && npm install && npm run dev            # :5173
```

Документы из backend автоматически синхронизируются в ML-индекс при старте
и при CRUD. LLM и эмбеддинги для локального запуска настраиваются через
`ml/.env` (см. `ml/README.md`).

Детали — в README соответствующих директорий (`backend/`, `ml/`, `frontend/`).
