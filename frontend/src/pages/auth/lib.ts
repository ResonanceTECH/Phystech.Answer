import { useQueryClient } from '@tanstack/react-query'
import { useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { sessionKey } from '@/entities/session'
import { isAdminRole, useAuthStore, type User } from '@/entities/user'

// Куда вести пользователя после входа: админы и эксперты — в админку,
// сотрудники — в план ознакомления.
export function homeFor(user: User): string {
  return isAdminRole(user.role) ? paths.admin.root : paths.app.onboarding
}

/**
 * После входа/регистрации: обновить общую сессию (её читают guards, шапка,
 * меню) и вернуть пользователя туда, откуда его отправили на вход (?next=).
 */
export function useAfterAuth() {
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  return (user: User) => {
    qc.setQueryData(sessionKey, user)
    void qc.invalidateQueries({ queryKey: ['documents'] })
    const next = params.get('next')
    navigate(next && next.startsWith('/') && !next.startsWith('//') ? next : homeFor(user), { replace: true })
  }
}

// Если сессия уже активна, страницы входа/регистрации сразу ведут в приложение.
export function useRedirectIfAuthed() {
  const fetchMe = useAuthStore((s) => s.fetchMe)
  const afterAuth = useAfterAuth()

  useEffect(() => {
    let alive = true
    fetchMe().then((user) => {
      if (alive && user) afterAuth(user)
    })
    return () => {
      alive = false
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fetchMe])
}
