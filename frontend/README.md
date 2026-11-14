# Phystech.Answer — фронтенд

React 19 · TypeScript · Vite · MUI · Zustand · TanStack Query · React Router · React Hook Form + Zod · Vitest

## Запуск

```bash
npm install
npm run dev      # http://localhost:5173
npm run build    # tsc -b && vite build
npm run lint     # oxlint
npm test         # vitest run
```

## Подключение к бэкенду

Запросы идут на `/api` и проксируются Vite на Go-backend `http://localhost:8080`
(см. `server.proxy` в `vite.config.ts`). Переопределяется через
`VITE_API_URL` — пример в `.env.example`.

Переменные окружения (`import.meta.env`, см. `shared/config/env.ts`):

| Переменная        | По умолчанию | Назначение                                        |
| ----------------- | ------------ | ------------------------------------------------- |
| `VITE_API_URL`    | `/api`       | Базовый URL API                                   |
| `VITE_APP_NAME`   | `Физтех.Ответ` | Название приложения в шапке/сайдбаре            |
| `VITE_DEMO_MODE`  | `true`       | Показывать бейдж «Демо» (авторизации нет)         |
| `VITE_ENABLE_ADMIN` | `true`     | Показывать раздел «Администрирование»            |

## Архитектура

Упрощённый Feature-Sliced Design. Слои сверху вниз:

```
src/
  app/            инициализация: providers (QueryClient + MUI theme + CssBaseline),
                  router + routePaths, layouts (App / Chat / Admin), theme
  pages/          экраны: home, chat, documents, document-details, instructions,
                  admin/{dashboard,documents,upload,stub}, not-found
  widgets/        крупные UI-блоки: app-sidebar, app-header, chat-history,
                  chat-workspace, sources-panel, documents-table, document-card,
                  admin-sidebar
  features/       действия: send-message, search-documents, save-instruction,
                  manage-document (CRUD-диалог), upload-documents
  entities/       доменные сущности: document, conversation, instruction,
                  source-reference, answer, user — типы, API, хуки, сторы
  shared/         api client (fetch + ApiError), queryKeys, config/env,
                  ui-kit (PageHeader, EmptyState, ErrorState, LoadingState,
                  StatusChip, ConfirmDialog), lib (даты, zustand-сторы UI)
```

Правила:

- Весь серверный трафик — только через типизированный `shared/api/client.ts`
  и `entities/*/api`. Компоненты не содержат бизнес-логики документов.
- Серверное состояние — TanStack Query (`entities/document`, `shared/api/queryKeys`);
  локальное — Zustand (`entities/conversation`, `entities/instruction`,
  `shared/lib/stores`).
- История диалогов и сохранённые инструкции хранятся в localStorage
  (`phystech.conversations.v1`, `phystech.instructions.v1`).
- Разделы админки без backend-API помечены заглушками (`pages/admin/stub`).
  Авторизация и разграничение прав на backend пока отсутствуют —
  доступ к `/admin` контролируется только флагом `VITE_ENABLE_ADMIN`,
  реальную защиту должен обеспечивать backend.

## Тесты

Unit-тесты на Vitest покрывают: вывод статуса ответа, маппинг источников,
сторы диалогов и инструкций, обработку ошибок API, утилиты загрузки файлов.

## Известные ограничения (MVP)

- Ответы `/api/ask` — REST, без SSE: статус «печатает…» имитируется pending-сообщением.
- Загрузка документов — только `.txt`/`.md`; PDF/DOCX требует серверного парсинга.
- «Обработка», «Редакции», «Противоречия», «Аналитика» в админке — placeholder-страницы.
