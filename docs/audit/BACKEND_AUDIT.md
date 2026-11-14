# BACKEND_AUDIT — Phystech.Answer

Go + net/http ServeMux + pgx/v5 + embedded migrations (0001–0004) + worker-goroutine в server (`cmd/server`→`jobs.Runner`; `cmd/worker` для отдельного масштабирования). Единый формат ошибок `{error:{code,message,retryable}}`.

## API-контракт (факт vs требуемый)

Реализовано: auth/*, users/me, departments, documents CRUD+versions/sections/relations/attachments/file/content/chunks/sources, search, users, conversations CRUD+messages, runs(+events SSE,answer,cancel), feedback, saved-instructions, action-routes, admin(upload,processing,retry,publish,reprocess,relations,dashboard), legacy `/api/*`.

Отсутствуют vs спецификация: `GET /admin/users` (есть `GET /users` admin-only — эквивалент), `GET /admin/versions`, `GET /admin/conflicts`, `GET /admin/documents` (list через `/documents`), admin audit-logs endpoint, user block/unblock, session list/revoke-self, change-password endpoint (только через reset-flow).

## AuthN/AuthZ (после фиксов аудита)

- Cookie `fiztech_session` HttpOnly+SameSite=Lax+Secure(https); в БД sha256 токена.
- Argon2id (+bcrypt legacy); reset-токены hashed+TTL+single-use+revoke.
- Rate-limit login/register/forgot/reset; generic forgot-ответ; `originCheck` CSRF; CORS whitelist.
- **Mass assignment**: `PATCH /users/me` → `model.ProfilePatch` без `role` — JSON-ключи вне whitelist отбрасываются. Проверено: `{"role":"admin"}` → role не меняется. ✅
- **Права на каждом endpoint**: conversations/runs — owner (404 чужим, admin/expert — доступ); documents mutations — admin; users — admin; restricted docs — privileged only; saved-instructions/action-routes — answer-ownership.

## PostgreSQL

- Полная схема: users/sessions/reset_tokens/departments/documents/versions/sections/chunks/source_files/jobs/conversations/messages/answer_runs/run_events/answers/steps/citations/relations/conflicts/feedback/saved_instructions/action_routes/audit_logs/user_doc_reads.
- FTS: `to_tsvector('russian')` на лету + `0002_fts_lower`; уникальности: email, login(nullable), source_files.sha256.
- Миграции embedded, `schema_migrations` версионирует. Данные не удалялись.

## Worker

- Claim `FOR UPDATE SKIP LOCKED`, concurrency, retry+backoff, stale-requeue 10мин, идемпотентность `answer:{run_id}`.
- `document.parse`: storage→extract→SetVersionContent(sections+chunks)→auto-published→ML index→detectRelations(pending).
- `answer.generate`: run events→ML ask→buildAnswer(status/steps/citations)→persist+assistant message.
- Гонок по одной версии нет (job per version, status transitions атомарны).

## Замечания

- `getJob` — linear scan ListJobs(500) (P2). `dashboard` — только docs+jobs (P2). `createUser` — пустой password_hash (P2). `Content-Disposition` filename без escape (P2). Rate-limit in-memory (P2). `answerFeedback` без RL (P3). Upload→auto-publish без review-гейта — процессный вопрос (P2).
- Audit log пишется для publish/reprocess/retry — покрытие частичное (загрузка не логируется отдельным событием — есть uploaded_by в source_files).
