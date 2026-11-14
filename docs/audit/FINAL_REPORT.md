# FINAL_REPORT — комплексный аудит Физтех.Ответ

Дата: 2026-11-14. Метод: статический анализ + живые E2E (curl API, psql, ML health) + unit/build тесты. Артефакты: `docs/audit/*.md`.

## 1. Что проверено

- Репозиторий: frontend (FSD React19), backend (Go+pgx+worker), ml (Go RAG+llama.cpp local), 4 миграции PG, compose, openapi, env, auth/documents/ml модули.
- Все HTTP-эндпоинты backend+ML прочитаны и сопоставлены с роутером и UI-вызовами.
- Живые тесты: auth-матрица anon/employee/admin, IDOR cross-user, conversation→run→ML-answer, upload→parse→publish→search→AI-citation, instructions/routes/feedback, mass-assignment, no-answer, relations 5990-1.

## 2. Что работало изначально

- Auth-цепочка полностью (register/login/logout/me/forgot/reset/onboarding/profile) — реальные, secure.
- ML/RAG реально: 28 docs/1568 chunks, локальные GGUF (llama.cpp host), structured answers с citations/steps/missing, честные статусы.
- Upload-пайплайн: multipart→dedup→job→parse→chunks→ML-index→relations.
- Worker: SKIP LOCKED/retry/backoff/stale/идемпотентность.
- Persistent answer-модель в PG + API.
- Compose: healthchecks, 127.0.0.1-порты, Mailpit.

## 3. Что было сломано

- **8 P0-уязвимостей доступа** (полный IDOR + анонимные мутации + утечка PII + restricted-leak) — подтверждены живыми запросами.
- **UI↔API разрыв**: chat/instructions на legacy `/api/ask`+localStorage; persistent API не подключён.

## 4. Что исправлено

- Все P0 → commit `362f5f2` (requireUser/requireAdmin/owner-checks, restricted-фильтры, session-user markRead, answer-ownership, `store.GetAnswer`). Проверено curl: anon 401, cross-user 404, employee 403, admin 200.
- Тестовый мусор удалён.

## 5. Изменённые файлы

Backend: `httpapi/{conversations,documents,users,runs,extras,search,ask}.go`, `store/{store,postgres}.go` (commit 362f5f2). Новое: `docs/audit/*.md`.

## 6–8. Тесты

| Проверка | Результат |
|---|---|
| go build/vet/test backend | ✅ clean, tests ok |
| Vitest frontend | ✅ 25/25 (без изменений frontend) |
| tsc+prod build | ✅ |
| E2E curl/psql сценарии | ✅ 20 в этом аудите + auth ранее |
| Падающие | нет |

## 9. Оставшиеся проблемы

- **P1**: frontend не использует persistent conversations/instructions/routes API (BUG-09/10).
- **P2**: admin stubs, dashboard частичный, auto-publish без review, OCR runtime отсутствует, getJob linear scan, rate-limit in-memory, PDF/DOCX parse не проверен.
- **P3**: filename-escape, feedback RL, bundle size.
- **Не измерено**: ML-метрики качества (нет eval-набора), UI-скриншоты (нет browser tool), OCR scan-тест.

## 10. До защиты

**Сотрудник (критично):**
1. ChatPage → `/conversations` + `/runs/{id}/events`(SSE) + `/runs/{id}/answer` → появятся answer_id/steps/citations.
2. «Сохранить инструкцию» → `POST /saved-instructions`; InstructionsPage → `GET /saved-instructions`; routes → `/action-routes`.

**Администратор (критично):**
3. `/admin/processing` → реальный список jobs (API готов); versions/conflicts/users/analytics — реализовать или скрыть.
4. Загрузить PDF/DOCX и прогнать parse (бинарные форматы не проверены).

**Желательно:**
5. Dashboard: users/AI-queries/unanswered/conflicts.
6. Eval-набор вопросов для метрик качества ML (10–15 размеченных Q&A из регламентов).
7. Playwright-прогон UI (desktop/tablet/mobile скриншоты).

## Готовность

- **Backend+ML+PG — готов к защите**: реальный RAG, цитаты, статусы, безопасность.
- **Frontend — частично**: auth/onboarding/profile/documents/upload/admin-dashboard реальные; chat/instructions — legacy.
- Оба пути **проходят через API** (доказано); «через UI» полностью — после пунктов 1–4.
