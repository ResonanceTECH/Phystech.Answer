# FRONTEND_AUDIT — Phystech.Answer

Стек: React 19 + TS + Vite, MUI, RHF+Zod, Zustand, motion/react, Vitest. FSD-структура. Дизайн: тёмный violet-noir лендинг (`landing.css`, `pl-*` классы, Unbounded/Onest) — он и есть фирменный стиль проекта (спека с #315DD6/Manrope относится к другому дизайну и НЕ применялась; локальные токены согласованы и сохранены).

## Маршруты (`app/router.tsx`)

Реализовано: `/ /login /register /forgot-password /reset-password /onboarding /app /app/profile /app/documents[/:id] /app/instructions /app/chat[/new|/:id] /admin /admin/documents /admin/documents/upload` + `AdminStubPage` на `/admin/processing|versions|conflicts|analytics`.

Отсутствуют vs требуемый список: `/app/history`, `/app/routes/:id`, `/admin/profile`, `/admin/users`, `/admin/ml`, `/admin/audit-logs`. Заглушки admin-разделов показаны как «в разработке» — допустимо для MVP, но processing мог бы быть реальным (API есть).

## Auth и гарды

- `authStore`: user/status/sessionExpired; HttpOnly cookie, `credentials:'include'`; токены не в JS.
- Гарды PublicOnly/RequireAuth/RequireOnboarding/RequireAdmin(admin|expert). Прямой URL защищён; sessionExpired обрабатывается.
- Auth-страницы наследуют landing-дизайн (AuthShell+auth.css, Hallmark lint clean).

## Сотрудник

- ProfilePage/Onboarding — реальные API, loading/error/empty states.
- DocumentsPage/Details/Search — реальные `/documents`, `/search`.
- **ChatPage — legacy**: `useSendMessage→POST /api/ask` (sync); история в Zustand persist→localStorage. НЕ использует `/conversations`, `/runs`, SSE, `/answers` — серверной истории и answer_id нет.
- **InstructionsPage — localStorage** (`entities/instruction`); backend `/saved-instructions`, `/action-routes` не вызываются. Action routes UI отсутствуют.

## Админ

- Dashboard — реальные счётчики (docs+jobs). Documents — список. Upload — multipart, статусы queued/duplicate/rejected.
- processing/versions/conflicts/analytics — AdminStubPage. Users UI нет (API `GET /users` admin-only готов).

## UI/UX

- Loading/empty/error states есть; адаптивность MUI+CSS; lint warnings only; prod build OK; Vitest 25/25.
- Bundle >500kB warning.
- **Не проверено**: скриншоты desktop/tablet/mobile (нет browser-автоматизации в сессии) — рекомендуется Playwright-прогон.

## Главные пробелы

1. Chat → persistent conversations/runs/SSE/answers (backend готов).
2. Instructions/action-routes/feedback → реальные API.
3. Admin stubs → processing реализовать, versions/conflicts/users/analytics — по остатку времени.
