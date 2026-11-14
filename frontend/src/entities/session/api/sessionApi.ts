import { api } from '@/shared/api'
import type { SessionUser } from '../model/types'

export function fetchMe(): Promise<SessionUser> {
  return api.get<SessionUser>('/v1/auth/me')
}

export function login(login: string, password: string): Promise<SessionUser> {
  return api.post<SessionUser>('/v1/auth/login', { login, password })
}

export function logout(): Promise<void> {
  return api.post<void>('/v1/auth/logout')
}
