export type Visibility = 'active' | 'hidden' | 'archived'
export type TargetType = 'all' | 'department' | 'user'
export type AssignRequirement = 'required' | 'optional'

export interface Department {
  id: string
  name: string
  members?: number
}

export interface Participant {
  id: string
  name: string
  email: string
  login?: string
  role: 'employee' | 'admin' | 'expert'
  departmentId?: string
  departmentName?: string
  position?: string
  active: boolean
  createdAt: string
  requiredTotal: number
  requiredRead: number
  optionalTotal: number
  optionalRead: number
}

export interface Assignment {
  id: string
  documentId: string
  targetType: TargetType
  targetId?: string
  targetName?: string
  requirement: AssignRequirement
  dueDate?: string
  createdAt: string
}

export const VISIBILITY_LABEL: Record<Visibility, string> = {
  active: 'Активен',
  hidden: 'Скрыт',
  archived: 'В архиве',
}

export const ROLE_LABEL: Record<Participant['role'], string> = {
  employee: 'Сотрудник',
  admin: 'Администратор',
  expert: 'Эксперт',
}

export const FOLDERS = [
  'Финансы проектов',
  'Бюджет и финансовая структура',
  'Кадры и командировки',
  'ИТ и инфраструктура',
  'Кампус и мероприятия',
  'Формы и бланки',
  'Прочие документы',
]
