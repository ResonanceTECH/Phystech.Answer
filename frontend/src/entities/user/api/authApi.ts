import { api } from '@/shared/api/client'
import type { Department, ProfilePatch, User } from '../model/types'

export interface LoginInput {
  email: string
  password: string
  remember: boolean
}

export interface RegisterInput {
  firstName: string
  lastName: string
  email: string
  password: string
}

// Сессия живёт в HttpOnly-cookie — отдельный токен на клиенте не храним.
export const authApi = {
  login: (input: LoginInput) => api.post<User>('/v1/auth/login', input),
  register: (input: RegisterInput) => api.post<User>('/v1/auth/register', input),
  logout: () => api.post<void>('/v1/auth/logout'),
  me: () => api.get<User>('/v1/auth/me'),
  updateProfile: (patch: ProfilePatch) => api.patch<User>('/v1/users/me', patch),
  departments: () => api.get<Department[]>('/v1/departments'),
  forgotPassword: (email: string) =>
    api.post<{ message: string }>('/v1/auth/forgot-password', { email }),
  resetPassword: (token: string, newPassword: string) =>
    api.post<{ message: string }>('/v1/auth/reset-password', { token, newPassword }),
}
