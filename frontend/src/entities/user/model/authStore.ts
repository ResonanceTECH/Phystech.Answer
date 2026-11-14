import { create } from 'zustand'
import { authApi } from '../api/authApi'
import type { User } from './types'

type AuthStatus = 'unknown' | 'authenticated' | 'anonymous'

interface AuthState {
  user: User | null
  status: AuthStatus
  setUser: (user: User) => void
  fetchMe: () => Promise<User | null>
  logout: () => Promise<void>
}

export const useAuthStore = create<AuthState>()((set) => ({
  user: null,
  status: 'unknown',

  setUser: (user) => set({ user, status: 'authenticated' }),

  fetchMe: async () => {
    try {
      const user = await authApi.me()
      set({ user, status: 'authenticated' })
      return user
    } catch {
      set({ user: null, status: 'anonymous' })
      return null
    }
  },

  logout: async () => {
    await authApi.logout().catch(() => {})
    set({ user: null, status: 'anonymous' })
  },
}))
