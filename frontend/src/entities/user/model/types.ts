export type UserRole = 'employee' | 'admin' | 'expert'

export type EmployeeCategory = 'employee' | 'teacher' | 'researcher' | 'manager'

export interface User {
  id: string
  name: string
  firstName?: string
  lastName?: string
  email: string
  login?: string
  role?: UserRole
  employeeCategory?: EmployeeCategory | string
  departmentId?: string
  position?: string
  onboardingComplete: boolean
  emailVerifiedAt?: string
  createdAt: string
}

export interface ProfilePatch {
  firstName?: string
  lastName?: string
  employeeCategory?: EmployeeCategory
  departmentId?: string
  position?: string
  onboardingComplete?: boolean
}

export interface Department {
  id: string
  name: string
}

// Административный доступ: для хакатона admin и expert объединены (см. backend).
export function isAdminRole(role?: string): boolean {
  return role === 'admin' || role === 'expert'
}
