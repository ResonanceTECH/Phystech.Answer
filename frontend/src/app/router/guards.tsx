import { CircularProgress } from '@mui/material'
import { useEffect, type ReactNode } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { isAdminRole, useAuthStore, type User } from '@/entities/user'

// Куда вести пользователя после входа: админы и эксперты — в админку,
// сотрудники — в приложение.
export function homeFor(user: User): string {
  return isAdminRole(user.role) ? paths.admin.root : paths.app.home
}

// Целевой маршрут по состоянию профиля.
function destinationFor(user: User): string {
  return user.onboardingComplete ? homeFor(user) : paths.auth.onboarding
}

function AuthGate({ children }: { children: ReactNode }) {
  return (
    <div
      style={{
        minHeight: '100svh',
        display: 'grid',
        placeItems: 'center',
        background: '#f6f7f9',
      }}
    >
      {children}
    </div>
  )
}

// Статус сессии известен только после /auth/me — один запрос на монтирование.
function useSessionStatus() {
  const status = useAuthStore((s) => s.status)
  const refreshUser = useAuthStore((s) => s.refreshUser)
  useEffect(() => {
    if (status === 'unknown') void refreshUser()
  }, [status, refreshUser])
  return status
}

// Защищённая зона: без сессии → /login (запоминаем, куда шёл пользователь),
// без онбординга → /onboarding, без роли admin/expert → /app.
export function RequireAuth({ adminOnly = false }: { adminOnly?: boolean }) {
  const status = useSessionStatus()
  const user = useAuthStore((s) => s.user)
  const sessionExpired = useAuthStore((s) => s.sessionExpired)
  const location = useLocation()

  if (status === 'unknown') {
    return (
      <AuthGate>
        <CircularProgress aria-label="Проверяем сессию" />
      </AuthGate>
    )
  }
  if (!user) {
    const from = location.pathname + location.search
    return (
      <Navigate
        to={paths.auth.login}
        replace
        state={{
          from,
          reason: sessionExpired ? 'expired' : 'auth',
        }}
      />
    )
  }
  if (!user.onboardingComplete && location.pathname !== paths.auth.onboarding) {
    return <Navigate to={paths.auth.onboarding} replace />
  }
  if (adminOnly && !isAdminRole(user.role)) {
    return <Navigate to={paths.app.home} replace />
  }
  return <Outlet />
}

// Онбординг доступен только авторизованным; настроенных — в кабинет.
export function RequireOnboarding() {
  const status = useSessionStatus()
  const user = useAuthStore((s) => s.user)
  const location = useLocation()

  if (status === 'unknown') {
    return (
      <AuthGate>
        <CircularProgress aria-label="Проверяем сессию" />
      </AuthGate>
    )
  }
  if (!user) {
    return (
      <Navigate
        to={paths.auth.login}
        replace
        state={{ from: location.pathname, reason: 'auth' }}
      />
    )
  }
  if (user.onboardingComplete) {
    return <Navigate to={homeFor(user)} replace />
  }
  return <Outlet />
}

// Страницы входа/регистрации для уже авторизованных не имеют смысла.
export function PublicOnly() {
  const status = useSessionStatus()
  const user = useAuthStore((s) => s.user)
  const location = useLocation()

  if (status === 'unknown') {
    return (
      <AuthGate>
        <CircularProgress aria-label="Проверяем сессию" />
      </AuthGate>
    )
  }
  if (user) {
    const from = (location.state as { from?: string } | null)?.from
    const target =
      from && from.startsWith('/') && !from.startsWith('//')
        ? from
        : destinationFor(user)
    return <Navigate to={target} replace />
  }
  return <Outlet />
}
