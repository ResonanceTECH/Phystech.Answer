export type UserRole = 'employee' | 'admin' | 'expert'

export interface User {
  id: string
  name: string
  email: string
  login?: string
  role?: UserRole
  createdAt: string
}

// Административный доступ: для хакатона admin и expert объединены (см. backend).
export function isAdminRole(role?: string): boolean {
  return role === 'admin' || role === 'expert'
}
