import { api } from '@/shared/api/client'
import type { User } from '../model/types'

export interface LoginInput {
  login: string
  password: string
}

export interface RegisterInput {
  name: string
  login: string
  email?: string
  password: string
}

// Сессия живёт в HttpOnly-cookie — отдельный токен на клиенте не храним.
export const authApi = {
  login: (input: LoginInput) => api.post<User>('/v1/auth/login', input),
  register: (input: RegisterInput) => api.post<User>('/v1/auth/register', input),
  logout: () => api.post<void>('/v1/auth/logout'),
  me: () => api.get<User>('/v1/auth/me'),
  forgotPassword: (login: string) =>
    api.post<{ ok: boolean }>('/v1/auth/forgot-password', { login }),
}
