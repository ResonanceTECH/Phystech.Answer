import type { PlanDocument } from './types'

const DAY = 24 * 60 * 60 * 1000

export interface PlanGroups {
  required: PlanDocument[]
  optional: PlanDocument[]
  library: Map<string, PlanDocument[]>
}

/** Обязательные — сначала непрочитанные с ближайшим сроком; библиотека — по папкам. */
export function groupPlan(items: PlanDocument[]): PlanGroups {
  const byDue = (a: PlanDocument, b: PlanDocument) => {
    if (a.read !== b.read) return a.read ? 1 : -1
    return (a.dueDate ?? '9999').localeCompare(b.dueDate ?? '9999') || a.title.localeCompare(b.title)
  }
  const required = items.filter((i) => i.requirement === 'required').sort(byDue)
  const optional = items.filter((i) => i.requirement === 'optional').sort(byDue)
  const library = new Map<string, PlanDocument[]>()
  for (const i of items.filter((x) => x.requirement === 'library')) {
    const key = i.folder || 'Прочие документы'
    library.set(key, [...(library.get(key) ?? []), i])
  }
  return { required, optional, library: new Map([...library.entries()].sort(([a], [b]) => a.localeCompare(b))) }
}

export function progress(items: PlanDocument[]) {
  const total = items.length
  const read = items.filter((i) => i.read).length
  return { total, read, pct: total === 0 ? 100 : Math.round((read / total) * 100) }
}

/** «Новый» — издан за последние 90 дней. */
export function isNewDocument(d: PlanDocument, now = Date.now()): boolean {
  if (!d.documentDate) return false
  const t = Date.parse(d.documentDate)
  return !Number.isNaN(t) && now - t <= 90 * DAY && t <= now + DAY
}

/** «Обновлён» — документ изменился после того, как сотрудник с ним ознакомился. */
export function isUpdatedSinceRead(d: PlanDocument): boolean {
  if (!d.read || !d.readAt) return false
  return Date.parse(d.updatedAt) > Date.parse(d.readAt) + 1000
}

export type DueState = 'none' | 'ok' | 'soon' | 'overdue'

/** Срок ознакомления: просрочен, скоро (≤ 3 дней), в порядке. */
export function dueState(d: PlanDocument, now = Date.now()): DueState {
  if (!d.dueDate || d.read) return 'none'
  const left = Date.parse(d.dueDate) + DAY - now
  if (left < 0) return 'overdue'
  if (left <= 3 * DAY) return 'soon'
  return 'ok'
}
