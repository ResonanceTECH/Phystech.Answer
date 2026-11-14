# SECURITY_AUDIT — Phystех.Answer

Проверено статически + живыми curl-запросами против `localhost:8080`. Секреты не включены.

## Исправлено в ходе аудита (commit 362f5f2, проверено curl)

| # | Уязвимость | Было | Стало | Проверка |
|---|---|---|---|---|
| P0-1 | IDOR conversations | аноним create/read/rename/delete/post в любой диалог | requireUser+owner→404 | anon 401, cross-user 404 |
| P0-2 | IDOR runs/answers/SSE | GET runs/events/answer/cancel без проверок | owner через run→conv→user | 404 чужому |
| P0-3 | Анонимные мутации каталога | POST/PUT/DELETE /documents без auth (документ «HACK» создан вживую) | mutations→admin, reads→user+restricted | 401/403 |
| P0-4 | Утечка пользователей | GET /users публично с email | admin-only | 401/403/200 |
| P0-5 | Restricted docs | list/get отдавали restricted всем | фильтр privileged | код+curl |
| P0-6 | markRead чужим именем | userId из body | session-user, чужой только admin | код |
| P0-7 | Answer-IDOR | saved-instructions/action-routes чужой answerId | GetAnswer→owner→404 | код |
| P0-8 | Анонимный ML | /api/ask, /search без auth | requireUser | 401 |

## Проверено — безопасно

- **Mass assignment**: `PATCH /users/me` whitelist `ProfilePatch` без role; `{"role":"admin"}` игнорируется (role остался employee). ✅
- Пароли Argon2id(+bcrypt legacy), PHC-тест; сессии sha256 в БД; cookie HttpOnly/Lax/Secure-conditional; reset hashed+TTL+single-use+revoke; generic forgot.
- Rate-limit auth-эндпоинтов + unit-test; `originCheck` CSRF; CORS whitelist (credentials только с origin).
- Upload: admin-only, ext-whitelist, SHA-256 dedup, 64MiB cap, content-addressed storage.
- SQL: pgx parameterized везде; FTS terms sanitized (`termSanitizer`).
- **ML-конфиденциальность**: LLM/embed/rerank — локальные llama.cpp (`host.docker.internal:8081-8083`), внешних API нет. ✅
- Порты compose → 127.0.0.1; `.env` не коммитится; recoverer без stack-trace утечек.

## Оставшиеся замечания

- `Content-Disposition` filename без URL-escape (P2, Go отсекает invalid bytes).
- `createUser` — пустой password_hash (P2).
- Rate-limit in-memory, не per-replica (P2).
- `/api/ask` sync — длинные соединения (P2, persistent runs есть).
- `answerFeedback` без rate-limit — спам (P3).
- Seed-учётки admin@example.com/admin123 — dev only, `SEED_DEMO_DATA` off для prod.
