// План ознакомления сотрудника (GET /api/v1/me/plan): обязательные документы
// (назначены всем, отделу или лично, могут иметь срок), рекомендуемые (KPI)
// и библиотека — остальные активные документы базы.

export type Requirement = 'required' | 'optional' | 'library'
export type PlanSource = 'all' | 'department' | 'user' | 'library'

export interface PlanDocument {
  documentId: string
  requirement: Requirement
  source: PlanSource
  dueDate?: string
  assignedAt?: string
  title: string
  documentNumber?: string
  documentType?: string
  documentDate?: string
  folder?: string
  updatedAt: string
  read: boolean
  readAt?: string
}

export interface PlanResponse {
  department?: string
  items: PlanDocument[]
}

export const SOURCE_LABEL: Record<PlanSource, string> = {
  all: 'всем сотрудникам',
  department: 'вашему подразделению',
  user: 'лично вам',
  library: '',
}
