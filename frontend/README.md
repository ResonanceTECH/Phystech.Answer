# Frontend

SPA «Физтех.Ответ» — React-приложение: лендинг, личный кабинет с чатом
вопрос–ответ по регламентам, каталог документов, сохранённые инструкции
и админ-раздел. Общение с backend идёт по HTTP API с cookie-сессией.

## Стек

- **React 19 + TypeScript**, сборка — **Vite 8** (`@vitejs/plugin-react`).
- **MUI 9** + Emotion — компоненты и тема.
- **TanStack Query 5** — серверное состояние (кэш, дедупликация, ретраи).
- **Zustand 5** — клиентское состояние (auth, диалоги, инструкции).
- **React Router 7** (`createBrowserRouter`) — маршруты и guard'ы.
- **React Hook Form + Zod** — формы и валидация.
- **Motion** — анимации (общие пресеты в `shared/lib/motionPresets.ts`).
- **Vitest** — unit-тесты, **Oxlint** — линтер.

## Команды

```bash
npm install
npm run dev        # vite dev-сервер, /api → http://localhost:8080
npm run build      # tsc -b && vite build → dist/
npm run lint       # oxlint
npm test           # vitest run
npm run preview    # предпросмотр сборки
```

## Конфигурация

Настройки — build-time переменные `VITE_*` (попадают в бандл, поэтому
не являются секретами; единая точка чтения — `src/shared/config/env.ts`):

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `VITE_API_URL` | `/api` | Базовый URL backend API |
| `VITE_APP_NAME` | `Физтех.Ответ` | Отображаемое имя продукта |
| `VITE_DEMO_MODE` | `true` | Демо-режим: бейдж и подсказки на экране входа |
| `VITE_ENABLE_ADMIN` | `true` | Показывать раздел администрирования в навигации |

`VITE_ENABLE_ADMIN` управляет только видимостью навигации — реальный
доступ к `/admin/*` проверяется guard'ом и backend.

В dev-режиме Vite проксирует `/api` на `http://localhost:8080`
(`changeOrigin: false`, чтобы backend видел исходный `Host` при
проверке Origin для cookie-сессий).

## Архитектура

Упрощённый Feature-Sliced Design — слои `app → pages → widgets →
features → entities → shared`, импорты только сверху вниз, алиас `@/`
указывает на `src/`.

```
src/
  app/            — providers, layouts, router (routePaths, guards), стили
  pages/          — экраны: home, auth/*, chat, documents,
                    document-details, instructions, profile,
                    landing, not-found, admin/*
  widgets/        — компоновка экранов: app-header, app-sidebar,
                    chat-history, chat-workspace, sources-panel,
                    document-card, documents-table, landing,
                    admin-sidebar
  features/       — сценарии: auth (формы входа/регистрации),
                    send-message, search-documents, manage-document,
                    upload-documents, save-instruction
  entities/       — доменные модели и их api/model:
                    user (+authStore, authApi), document (+api),
                    conversation (conversationStore), answer,
                    source-reference, instruction (instructionStore),
                    landing (контент)
  shared/         — api/client, config/env, lib (cn, dates, scroll,
                    stores, motionPresets), ui (карточки состояний,
                    StatusChip, PageHeader, ConfirmDialog…)
```

119 ts/tsx-файлов. Сторонних UI-фреймворков поверх MUI нет — всё
собрано на компонентах MUI и `shared/ui`.

## Маршрутизация и защита

Роутер — `src/app/router.tsx`, все пути централизованы в
`routePaths.ts`.

| Зона | Guard | Пути |
|---|---|---|
| Публичные auth-страницы | `PublicOnly` (авторизованных → домой) | `/login`, `/register`, страницы восстановления доступа |
| Онбординг | `RequireOnboarding` | `/onboarding` |
| Приложение | `RequireAuth` | `/app`, `/app/profile`, `/app/chat/new`, `/app/chat/:conversationId`, `/app/documents`, `/app/documents/:documentId`, `/app/instructions` |
| Админка | `RequireAuth adminOnly` | `/admin` (+ `documents`, `documents/upload`, `processing`, `versions`, `conflicts`, `analytics`) |
| Лендинг | — | `/` |
| 404 | — | `*` |

Логика guard'ов (`src/app/router/guards.tsx`):

- Статус сессии при монтировании `unknown` → один запрос
  `GET /v1/auth/me`, до ответа — спиннер.
- Нет пользователя → редирект на `/login` с `state.from` (куда шёл)
  и `reason` (`expired` при протухшей сессии).
- `onboardingComplete === false` → принудительный `/onboarding`.
- `adminOnly` проверяет `isAdminRole(role)` — `admin` и `expert`
  считаются административными; остальных отправляет в `/app`.
- `homeFor(user)`: admin/expert → `/admin`, остальные → `/app`.

## Состояние

- **Серверное** — TanStack Query (`shared/api/queryKeys.ts` для ключей).
  Мутации инвалидируют связанные запросы; GET'ы кэшируются.
- **Auth** — `entities/user/model/authStore.ts` (Zustand):
  `status` (`unknown|...`), `user`, `sessionExpired`, действия
  `login/register/logout/refreshUser/updateProfile` поверх `authApi`.
- **Диалоги** — `entities/conversation/model/conversationStore.ts`,
  персист в `localStorage` под ключом `phystech.conversations.v1`.
- **Инструкции** — `entities/instruction/model/instructionStore.ts`,
  персист под `phystech.instructions.v1` (ответы «сохранить как
  инструкцию» хранятся локально на устройстве).

## API-клиент

`shared/api/client.ts` — тонкая обёртка над `fetch`:

- `credentials: 'include'` — cookie-сессия ходит на тот же origin
  (в dev — через Vite-прокси).
- `BASE_URL = env.apiBaseUrl` → `/api`.
- Ошибки backend (`{error:{code,message,retryable}}`) нормализуются
  в `ApiError` (`shared/api/errors.ts`).

Используемые эндпоинты (текущий фронтенд работает в основном на
legacy-алиасах backend):

- `POST /api/ask` — синхронный вопрос (чат, `sendMessage`).
- `GET/POST/PUT/DELETE /api/documents*` — каталог и CRUD документов
  (включая админ-страницы).
- `GET /api/users`, `POST /api/users/{id}/unread` — список и метки
  прочитанного.
- `/api/v1/auth/{login,register,logout,me}`, `PATCH /api/v1/users/me`,
  `GET /api/v1/departments` — auth и профиль.

Versioned-эндпоинты `/api/v1/conversations`, `/api/v1/runs/*` (SSE),
`/api/v1/saved-instructions`, `/api/v1/action-routes` существуют на
backend, но UI их пока не использует — см. `api/README.md`.

## Основные сценарии

### Чат (`pages/chat`, `widgets/chat-workspace`, `features/send-message`)

- Новый диалог — `/app/chat/new` (сообщение можно передать через
  router state и отправить автоматически); существующий —
  `/app/chat/:conversationId`.
- `useSendMessage`: оптимистичное pending-сообщение →
  `POST /api/ask {question}` → ответ мапится в `Answer`
  (status, sources, warnings) → сообщение ассистента в стор диалога.
- `MessageBubble`: статус-чипы ответа (`answered`, `partial`,
  `not_found`, `conflict`, `clarification_required`), список источников,
  пометка degraded-режима, кнопка «сохранить как инструкцию»
  (`features/save-instruction`).
- `SourcesPanel` (виджет): клик по источнику открывает панель с
  цитатой и ссылкой на документ.
- История диалогов — `widgets/chat-history`; список диалогов
  персистится в localStorage (backend-модель conversations/runs пока
  не подключена к UI).

Важно: фронтенд потребляет плоский ответ `/api/ask`
(`{answer, sources, stub}`); богатый структурный payload ML
(`steps`, `citations`, `missing_information`) бэкенд сохраняет в
`answer.generate`-джобах, но текущий чат его не рендерит.

### Документы (`pages/documents`, `pages/document-details`, `features/search-documents`)

- Каталог — `GET /api/documents`; фильтр по названию/содержимому/тегам
  выполняется на клиенте, состояние фильтров — в URL query
  (`?q=`, `?tag=`), поэтому ссылки на выборки шарятся.
- Карточка документа — `widgets/document-card` (версия, теги, дата,
  бейдж прочитанного через `/api/users/{id}/unread`).
- Страница документа — метаданные, текст последней версии, кнопка
  «обсудить с ассистентом» (переход в чат с контекстом документа).

### Профиль и онбординг (`pages/profile`, `pages/auth/onboarding`)

- Онбординг: категория сотрудника, подразделение (из
  `GET /api/v1/departments`), должность → `PATCH /api/v1/users/me`.
- Профиль: те же поля + редактирование имени; role не редактируется.

### Инструкции (`pages/instructions`)

Локальный список сохранённых ответов ассистента
(`instructionStore`, localStorage) — CRUD без backend.

### Админка (`pages/admin/*`, `widgets/admin-sidebar`)

- **Dashboard** (`/admin`) — счётчики документов/пользователей/тегов
  и список последних документов.
- **Документы** (`/admin/documents`) — таблица
  (`widgets/documents-table`) + диалог создания/редактирования через
  `manage-document` (legacy `POST/PUT /api/documents`).
- **Загрузка** (`/admin/documents/upload`) — читает текстовый файл
  на клиенте (`.txt`, `.md`, `.markdown`, `.text` — `readTextFile`)
  и создаёт документ через `createDocument`. PDF/DOCX/XLSX
  отклоняются с пояснением — backend-парсинг бинарных форматов есть,
  но UI-загрузка их пока не поддерживает (admin upload API
  `/api/v1/admin/documents/upload` с multipart тоже не задействован).
- **Заглушки**: `processing`, `versions`, `conflicts`, `analytics` —
  `AdminStubPage` с описанием «раздел в разработке».

## Тесты и качество

- `npm test` — Vitest, 8 файлов / 25 тестов: api-клиент и ошибки,
  authStore, conversationStore, instructionStore, source-reference,
  answer types, readTextFile и модули валидации форм авторизации
  (`features/auth/lib`).
- `npm run lint` — oxlint; сейчас 4 предупреждения
  (`incompatible-library` в формах auth, `only-export-components`
  в guards, `set-state-in-effect` в поиске) — без ошибок.
- `npm run build` — `tsc -b` + Vite; бандл ~1 МБ (gzip ~329 КБ),
  Vite предупреждает о chunk >500 КБ — кандидат на code-splitting.

## Ограничения и известные долги

- Чат работает на синхронном `/api/ask`; персистентные
  conversations/runs/SSE на backend реализованы, но не подключены.
- Ответ рендерится плоско — структурированные `steps`/`citations`
  из ML-payload не доходят до UI.
- Диалоги и инструкции — localStorage: нет синхронизации между
  устройствами и нет серверной истории.
- Админ-загрузка принимает только текстовые файлы.
- Четыре раздела админки — заглушки.
- `demoMode`/`enableAdmin` — только UX-флаги; безопасность на backend.
