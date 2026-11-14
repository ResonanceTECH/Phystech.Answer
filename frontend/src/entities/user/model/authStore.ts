import { create } from 'zustand'
import { authApi, type LoginInput, type RegisterInput } from '../api/authApi'
import type { ProfilePatch, User } from './types'

type AuthStatus = 'unknown' | 'authenticated' | 'anonymous'

interface AuthState {
  user: User | null
  status: AuthStatus
  // true, если /auth/me потерял ранее действовавшую сессию — для экрана «сессия истекла».
  sessionExpired: boolean

  // Действия. login/register бросают ApiError — формы показывают текст сами.
  login: (input: LoginInput) => Promise<User>
  register: (input: RegisterInput) => Promise<User>
  logout: () => Promise<void>
  refreshUser: () => Promise<User | null>
  updateProfile: (patch: ProfilePatch) => Promise<User>
}

export const useAuthStore = create<AuthState>()((set, get) => ({
  user: null,
  status: 'unknown',
  sessionExpired: false,

  login: async (input) => {
    const user = await authApi.login(input)
    set({ user, status: 'authenticated', sessionExpired: false })
    return user
  },

  register: async (input) => {
    const user = await authApi.register(input)
    set({ user, status: 'authenticated', sessionExpired: false })
    return user
  },

  logout: async () => {
    await authApi.logout().catch(() => {})
    set({ user: null, status: 'anonymous', sessionExpired: false })
  },

  refreshUser: async () => {
    try {
      const user = await authApi.me()
      set({ user, status: 'authenticated', sessionExpired: false })
      return user
    } catch {
      set({
        user: null,
        status: 'anonymous',
        sessionExpired: get().status === 'authenticated',
      })
      return null
    }
  },

  updateProfile: async (patch) => {
    const user = await authApi.updateProfile(patch)
    set({ user })
    return user
  },
}))
