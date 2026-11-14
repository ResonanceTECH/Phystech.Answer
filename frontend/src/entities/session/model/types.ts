// Сущность Session — текущий пользователь и его роль.

export type UserRole = 'employee' | 'admin' | 'expert'

export interface SessionUser {
  id: string
  name: string
  email: string
  login?: string
  role: UserRole
}

export const ROLE_LABEL: Record<UserRole, string> = {
  employee: 'Сотрудник',
  admin: 'Администратор',
  expert: 'Эксперт нормативной базы',
}

/** Администратор и эксперт управляют базой документов. */
export function canManageDocuments(user: SessionUser | null | undefined): boolean {
  return user?.role === 'admin' || user?.role === 'expert'
}

/** Демо-учётные записи (засеиваются бэкендом) — для быстрого входа на демо. */
export const DEMO_ACCOUNTS = [
  { login: 'ivan', password: 'ivan123', label: 'Сотрудник', hint: 'Иван Петров — новый сотрудник' },
  { login: 'admin', password: 'admin123', label: 'Администратор', hint: 'загрузка и контроль документов' },
] as const
