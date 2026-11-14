# Phystech.Answer — фронтенд

React 19 · TypeScript · Vite · Tailwind CSS v4 · React Router · TanStack Query

## Запуск

```bash
npm install
npm run dev      # http://localhost:5173
npm run build    # tsc -b && vite build
npm run lint     # oxlint
```

## Подключение к бэкенду

Запросы идут на `/api` и проксируются Vite на `http://localhost:8000`
(см. `server.proxy` в `vite.config.ts`). Переопределяется через
`VITE_API_URL` — пример в `.env.example`.

## Структура

```
src/
  api/client.ts      fetch-обёртка api.get/post/put/delete + ApiError
  api/types.ts       Document, AskResponse, AskSource
  app/providers.tsx  QueryClientProvider
  app/router.tsx     createBrowserRouter, маршруты
  components/        Layout и общие компоненты
  pages/
    ChatPage.tsx       ассистент: диалог, источники, история (localStorage)
    DocumentsPage.tsx  каталог документов с поиском
    DocumentPage.tsx   карточка документа + «Обсудить с ассистентом»
    NotFoundPage.tsx
  main.tsx           вход: Providers + RouterProvider
```

История диалога хранится в localStorage (`phystech.chat.v1`).

Алиас `@/` → `src/` (настроен в `vite.config.ts` и `tsconfig.app.json`).
