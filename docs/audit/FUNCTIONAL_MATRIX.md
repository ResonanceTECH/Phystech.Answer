# FUNCTIONAL_MATRIX — аудит Физтех.Ответ

Дата: 2026-11-14. Окружение: docker-compose (postgres:17, ml, backend:8080, frontend:3000, mailpit). Статусы: implemented / partially_implemented / mock_only / broken / not_implemented / not_tested.

| Функция | Frontend | Backend | ML | Сквозной тест | Статус |
|---|---|---|---|---|---|
| Регистрация | ✅ RegisterPage (RHF+Zod) | ✅ register, Argon2id, domain check, rate-limit | — | curl register→login→me | implemented |
| Авторизация | ✅ LoginPage, cookie session | ✅ login/logout/me, sessions в PG (sha256 token) | — | curl + IDOR-матрица | implemented |
| Восстановление пароля | ✅ forgot/reset pages | ✅ hashed token, TTL, single-use, revoke sessions, Mailpit | — | Mailpit E2E (ранее) | implemented |
| Профиль сотрудника | ✅ ProfilePage, OnboardingPage | ✅ PATCH /users/me (whitelist, без role), departments | — | curl PATCH + mass-assign test | implemented |
| Гарды/маршруты | ✅ PublicOnly/RequireAuth/RequireOnboarding/RequireAdmin | — | — | code review | implemented |
| AI-ассистент (диалог) | 🟡 ChatPage→`/api/ask` sync, история в localStorage | ✅ conversations/runs/answers/citations/SSE — полный persistent API | ✅ RAG: retrieve+rerank+LLM+verify | curl: run→answer со sources | partially_implemented (UI на legacy-пути) |
| Каталог документов | ✅ DocumentsPage/Details/Search | ✅ list/get + restricted-фильтр, FTS /search | — | curl employee | implemented |
| Управление документами | ✅ upload page | ✅ upload(multipart,SHA-dedup,64MiB), CRUD admin-only, versions, file stream | index push | curl upload .txt→published | implemented |
| Парсинг PDF/DOCX/XLSX/TXT/MD | — | ✅ worker document.parse, internal/parse | — | .txt E2E; pdf/docx/xlsx — not_tested | partially_implemented |
| OCR (сканы) | — | ⚠️ ErrNoText→requires_review; runtime OCR нет, `/internal/v1/ocr`→501 | offline pipeline only | — | not_implemented (runtime) |
| Индексация/чанкинг | — | ✅ SetVersionContent→sections+chunks | ✅ chunk+embed в ml store | health 28 docs/1568 chunks | implemented |
| Гибридный поиск+rerank | 🟡 search field (backend FTS) | ✅ /search tsvector russian | ✅ vector+rerank в RAG | ask E2E с релевантными sources | implemented |
| Цитирование | 🟡 sources panel | ✅ citations→version→chunk resolve | ✅ chunk-id aliases | answer JSON: title+section+quote+score | implemented (частично в UI) |
| Пошаговые инструкции | ❌ localStorage | ✅ answers.steps + saved-instructions + action-routes API | ✅ structured steps | curl create/list | partially_implemented |
| Редакции/версии | ❌ versions stub | ✅ versions API, relations (repeals/amends/…), verify, detector в worker | relations LLM extract | 5990-1→repeals confirmed | partially_implemented |
| Админ-дашборд | ✅ AdminDashboardPage | 🟡 docs+jobs counts only | — | curl admin | partially_implemented |
| Пользователи (admin) | ❌ stub | ✅ GET/POST /users admin-only | — | curl 403/200 | partially_implemented |
| Аналитика/ML-monitoring/audit-logs UI | ❌ stub/нет маршрутов | 🟡 audit_logs пишутся, endpoint просмотра нет | — | — | not_implemented |
| Обработка jobs | 🟡 processing stub | ✅ list/get/retry, SKIP LOCKED, backoff, stale-requeue | — | jobs stats в dashboard | partially_implemented |
| Конфликты норм | ❌ stub | 🟡 document_conflicts таблица, relations pending | — | — | not_implemented |
| Rate-limit/CSRF/CORS | — | ✅ auth endpoints RL, originCheck, CORS whitelist | — | unit+code | implemented |
| IDOR/права | — | ✅ owner-checks везде (фикс аудита) | — | curl cross-user 404 | implemented |

## Ключевой вывод

Backend+ML+PG — реальный и проверенный стек. Разрыв: frontend chat/instructions на legacy `/api/ask`+localStorage вместо persistent API; админ-разделы versions/conflicts/analytics/users — заглушки; OCR runtime отсутствует.
