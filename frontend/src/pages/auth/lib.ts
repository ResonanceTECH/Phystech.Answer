import { useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { isAdminRole, useAuthStore, type User } from '@/entities/user'

// Куда вести пользователя после входа: админы и эксперты — в админку,
// сотрудники — в приложение.
export function homeFor(user: User): string {
  return isAdminRole(user.role) ? paths.admin.root : paths.app.home
}

// Если сессия уже активна, страницы входа/регистрации сразу ведут в приложение.
export function useRedirectIfAuthed() {
  const navigate = useNavigate()
  const fetchMe = useAuthStore((s) => s.fetchMe)

  useEffect(() => {
    let alive = true
    fetchMe().then((user) => {
      if (alive && user) navigate(homeFor(user), { replace: true })
    })
    return () => {
      alive = false
    }
  }, [fetchMe, navigate])
}
