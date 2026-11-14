# BUG_REPORT — Физтех.Ответ

## P0 — исправлены (commit 362f5f2, проверено curl)

| ID | Баг | Фикс | Статус |
|---|---|---|---|
| BUG-01 | IDOR conversations: аноним/чужой create-read-rename-delete-post | requireUser+requireConversationAccess→404 | ✅ verified |
| BUG-02 | IDOR runs/events/answer/cancel | requireRunAccess (run→conv→user) | ✅ verified |
| BUG-03 | Анонимные мутации /documents (документ создан вживую) | mutations→admin, reads→user+restricted | ✅ verified |
| BUG-04 | GET /users публично с PII | admin-only | ✅ verified |
| BUG-05 | restricted docs видны всем | privileged-фильтр | ✅ verified |
| BUG-06 | markRead чужим userId | session-user | ✅ verified |
| BUG-07 | Чужой answerId в instructions/routes | GetAnswer→owner→404 | ✅ verified |
| BUG-08 | Анонимные /api/ask, /search | requireUser | ✅ verified |

## P1 — архитектурный разрыв (open)

| ID | Баг | Влияние | Статус |
|---|---|---|---|
| BUG-09 | ChatPage→`/api/ask`+localStorage, не `/conversations`+SSE | нет серверной истории, answer_id, steps-UI, «проверяемой инструкции» | ⚠️ open — нужен frontend |
| BUG-10 | Instructions/action-routes/feedback → localStorage/нет UI | backend API работает, UI не подключён | ⚠️ open |

## P2 — дефекты/риски (open)

| ID | Описание |
|---|---|
| BUG-11 | `getJob` linear scan ListJobs(500) |
| BUG-12 | dashboard только docs+jobs (нет users/AI-queries/conflicts) |
| BUG-13 | `createUser` пустой password_hash |
| BUG-14 | Content-Disposition filename без escape |
| BUG-15 | Rate-limit in-memory (не per-replica) |
| BUG-16 | answerFeedback без rate-limit |
| BUG-17 | AdminStubPage: processing/versions/conflicts/analytics |
| BUG-18 | Нет UI: admin users, audit-logs, ML-monitoring, /app/history, /app/routes/:id |
| BUG-19 | Upload→auto-publish без review-гейта (процессный вопрос) |
| BUG-20 | OCR runtime отсутствует → сканы в requires_review навсегда |
| BUG-21 | PDF/DOCX/XLSX parse не проверен E2E |

## Исправлено ранее (прошлые сессии)

Argon2id PHC-парсинг; /me ambiguous-SQL 401; users.login UNIQUE DEFAULT ''; mailer Cyrillic; CORS `*`+credentials; Vite proxy changeOrigin; password-strength; Zod v4; TS-ошибки.
