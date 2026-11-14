import type { Document } from '@/entities/document'

// План ознакомления нового сотрудника: документы базы раскладываются по
// модулям (неделям) по ключевым словам названия. Новый документ попадает
// в нужный модуль автоматически — без правки кода.

export interface PlanModule {
  id: string
  week: string
  title: string
  description: string
  keywords: RegExp
}

export const PLAN_MODULES: PlanModule[] = [
  {
    id: 'workplace',
    week: 'Неделя 1',
    title: 'Рабочее место и сервисы',
    description: 'Техподдержка, ИТ-инфраструктура, общежитие и площадки для мероприятий.',
    keywords: /(техподдерж|технической поддерж|центра обработки|цод|поселени|общежит|зал)/i,
  },
  {
    id: 'people',
    week: 'Неделя 2',
    title: 'Кадры и командировки',
    description: 'Подбор персонала и служебные командировки.',
    keywords: /(подбор|персонал|командир)/i,
  },
  {
    id: 'projects',
    week: 'Неделя 3',
    title: 'Финансы проектов',
    description: 'Финансовые лицевые счета, сметы и внутреннее заимствование.',
    keywords: /(флс|лицев|смет|заимств|794-1|889-1|3065-1|5990-1|700)/i,
  },
  {
    id: 'budget',
    week: 'Неделя 4',
    title: 'Бюджет и отчётность',
    description: 'Формирование бюджета, финансовая структура, «Электронный бюджет».',
    keywords: /(бюджет|фин\.? ?структур|(?:^|[\s«])эб(?:$|[\s»])|3614-1|3699-1)/i,
  },
]

const FORM_RE = /(бланк|форм|приложени)/i

/** Формы и бланки — справочные материалы, в прогресс не входят. */
export function isReference(doc: Document): boolean {
  return doc.documentType === 'форма' || doc.documentType === 'приложение' || FORM_RE.test(doc.title.split(' ')[0] ?? '')
}

export interface PlannedModule extends PlanModule {
  docs: Document[]
  read: number
}

export interface Plan {
  modules: PlannedModule[]
  extra: Document[]
  reference: Document[]
  total: number
  read: number
  next?: Document
}

function haystack(d: Document) {
  return `${d.title} ${d.documentNumber ?? ''}`
}

export function buildPlan(docs: Document[]): Plan {
  const modules: PlannedModule[] = PLAN_MODULES.map((m) => ({ ...m, docs: [], read: 0 }))
  const extra: Document[] = []
  const reference: Document[] = []
  const byDate = [...docs].sort((a, b) => (a.documentDate ?? '').localeCompare(b.documentDate ?? ''))

  for (const d of byDate) {
    if (isReference(d)) {
      reference.push(d)
      continue
    }
    const m = modules.find((mod) => mod.keywords.test(haystack(d)))
    if (m) m.docs.push(d)
    else extra.push(d)
  }
  for (const m of modules) m.read = m.docs.filter((d) => d.read).length

  const required = [...modules.flatMap((m) => m.docs), ...extra]
  const read = required.filter((d) => d.read).length
  return {
    modules: modules.filter((m) => m.docs.length > 0),
    extra,
    reference,
    total: required.length,
    read,
    next: required.find((d) => !d.read),
  }
}

const DAY = 24 * 60 * 60 * 1000

/** «Новый» — документ издан за последние 90 дней. */
export function isNewDocument(doc: Document, now = Date.now()): boolean {
  if (!doc.documentDate) return false
  const t = Date.parse(doc.documentDate)
  return !Number.isNaN(t) && now - t <= 90 * DAY && t <= now + DAY
}

/** «Обновлён» — новая версия появилась после того, как сотрудник прочитал документ. */
export function isUpdatedSinceRead(doc: Document): boolean {
  if (!doc.read || !doc.readAt) return false
  return Date.parse(doc.updatedAt) > Date.parse(doc.readAt) + 1000
}
