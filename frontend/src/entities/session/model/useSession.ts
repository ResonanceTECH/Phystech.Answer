import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ApiError } from '@/shared/api'
import { fetchMe, login, logout } from '../api/sessionApi'
import type { SessionUser } from './types'

export const sessionKey = ['session'] as const

/** Текущий пользователь: null — не вошёл; undefined — ещё загружается. */
export function useSession() {
  const query = useQuery({
    queryKey: sessionKey,
    queryFn: async (): Promise<SessionUser | null> => {
      try {
        return await fetchMe()
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) return null
        throw err
      }
    },
    staleTime: 5 * 60_000,
    retry: false,
  })
  return { user: query.data, isLoading: query.isLoading, isError: query.isError }
}

export function useLogin() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ login: l, password }: { login: string; password: string }) => login(l, password),
    onSuccess: (user) => {
      qc.setQueryData(sessionKey, user)
      qc.invalidateQueries({ queryKey: ['documents'] })
    },
  })
}

export function useLogout() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: logout,
    onSettled: () => {
      qc.setQueryData(sessionKey, null)
      qc.removeQueries({ queryKey: ['documents'] })
    },
  })
}
