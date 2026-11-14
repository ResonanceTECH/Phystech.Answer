# E2E_RESULTS — Физтех.Ответ

Окружение: docker-compose (postgres/ml/backend:8080/frontend:3000/mailpit). Дата 2026-11-14. Аккаунты: seed ivan/maria/admin/expert + временные. Проверки через curl к реальному API + psql к БД.

## Сценарии (запущены)

| # | Сценарий | Результат | Доказательство |
|---|---|---|---|
| 1 | Анон→защищённые API (11 эндпоинтов) | ✅ все 401 | curl-матрица |
| 2 | Анон-мутации (до фикса создавал doc/conv) | ✅ после фикса 401 | `doc_bf6385406d394c11` создан до, удалён |
| 3 | Employee: conversation+question→run | ✅ queued→completed ~30s | `run_1c1e6e6b0441e42b` |
| 4 | ML answer: status/steps/sources/missing | ✅ partial, 3 sources, 1 step | `/runs/{id}/answer` |
| 5 | IDOR: чужой conversation read/post | ✅ 404 | curl maria→ivan |
| 6 | IDOR: чужой run read | ✅ 404 | curl |
| 7 | Employee→/users, /admin/dashboard | ✅ 403 | curl |
| 8 | Admin→dashboard/users | ✅ 200: 28 docs, jobs; 10 users | curl |
| 9 | Admin upload .txt→parse→published | ✅ ~10s, 4 chunks | `doc_9e64e6927290a896` |
| 10 | Publish endpoint | ✅ 200 | curl |
| 11 | Employee FTS-search нового doc | ✅ в индексе (rank ниже top-10 OR-запроса — ожидаемо) | psql 2 chunks match |
| 12 | Employee AI-вопрос → новый doc как источник | ✅ answered, источник №3 = загруженный приказ | `run_14b6454169e452b5` |
| 13 | saved-instruction/action-route/feedback | ✅ 201/201/200 persist | curl |
| 14 | ML /health + /api/ask напрямую | ✅ 28 docs/1568 chunks, real answer | curl |
| 15 | Restricted-доступ фильтрация | ✅ employee не видит restricted | код+curl |
| 16 | Mass-assignment `{"role":"admin"}` в PATCH /me | ✅ role не изменился | curl |
| 17 | ML-конфиденциальность | ✅ только локальные llama.cpp | config/compose |
| 18 | Редакции: 5990-1 от 03.11.2026 | ✅ repeals→«Порядок внутр. заимствования (ред.2024)» confirmed | psql |
| 19 | No-answer: «вид на жительство» | ✅ not_found, missing_information, без галлюцинаций | `run_5a978494cdf27cc3` |
| 20 | go build/vet/test backend | ✅ clean, tests ok | go test ./... |

## Переиспользовано из прошлых сессий (evidence)

- Register→login→me→departments→PATCH profile ✅; forgot→Mailpit→reset→revoke ✅; `/me` SQL-fix регрессии нет; Vitest 25/25; tsc+prod build; route smoke 200.

## Не прогонялось / ограничения

- **UI-клики/скриншоты**: browser-автоматизации в сессии нет — frontend проверен по коду+API, не визуально (Playwright рекомендован).
- **PDF/DOCX/XLSX parse E2E** — код есть, бинарные форматы не гонялись (.txt проверен).
- **OCR/скан-PDF** — runtime OCR отсутствует (501), сценарий 10 невозможен до реализации.
- **ML-метрики** (Recall@5/MRR/nDCG/Citation Accuracy/…) — нет размеченного eval-набора.
- **Сценарий 9 (отказ ML)** — stub-режим проверен кодом+ранее, живой failover не имитировался.
- Нагрузочное тестирование — не входило.
