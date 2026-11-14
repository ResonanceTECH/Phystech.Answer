import { Box, CircularProgress } from '@mui/material'
import type { ReactNode } from 'react'
import { Navigate, useLocation } from 'react-router-dom'
import { canManageDocuments, useSession } from '@/entities/session'
import { paths } from './routePaths'

function FullscreenLoader() {
  return (
    <Box sx={{ minHeight: '100svh', display: 'grid', placeItems: 'center' }}>
      <CircularProgress />
    </Box>
  )
}

/** Пускает только вошедших; остальных — на /login с возвратом обратно. */
export function RequireAuth({ children }: { children: ReactNode }) {
  const { user, isLoading } = useSession()
  const location = useLocation()
  if (isLoading) return <FullscreenLoader />
  if (!user) {
    const next = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`${paths.login}?next=${next}`} replace />
  }
  return <>{children}</>
}

/** Админ-раздел: администратор или эксперт; сотрудник — на главную приложения. */
export function RequireAdmin({ children }: { children: ReactNode }) {
  const { user, isLoading } = useSession()
  const location = useLocation()
  if (isLoading) return <FullscreenLoader />
  if (!user) {
    return <Navigate to={`${paths.login}?next=${encodeURIComponent(location.pathname)}`} replace />
  }
  if (!canManageDocuments(user)) return <Navigate to={paths.app.home} replace />
  return <>{children}</>
}
