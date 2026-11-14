import { describe, expect, it } from 'vitest'
import { dueState, groupPlan, isUpdatedSinceRead, progress } from './plan'
import type { PlanDocument } from './types'

const d = (p: Partial<PlanDocument>): PlanDocument => ({
  documentId: p.title ?? 'x',
  requirement: 'required',
  source: 'department',
  title: 'Документ',
  updatedAt: '2026-01-01T00:00:00Z',
  read: false,
  ...p,
})

describe('groupPlan', () => {
  it('делит на обязательные, рекомендуемые и библиотеку по папкам', () => {
    const g = groupPlan([
      d({ title: 'Б', dueDate: '2026-12-01' }),
      d({ title: 'А', dueDate: '2026-11-20' }),
      d({ title: 'Прочитан', read: true }),
      d({ title: 'KPI', requirement: 'optional' }),
      d({ title: 'Л1', requirement: 'library', folder: 'Финансы проектов' }),
      d({ title: 'Л2', requirement: 'library' }),
    ])
    expect(g.required.map((x) => x.title)).toEqual(['А', 'Б', 'Прочитан'])
    expect(g.optional).toHaveLength(1)
    expect([...g.library.keys()]).toEqual(['Прочие документы', 'Финансы проектов'])
  })
})

describe('прогресс и сроки', () => {
  const now = Date.parse('2026-11-14T12:00:00Z')
  it('считает прочитанные', () => {
    expect(progress([d({ read: true }), d({})])).toEqual({ total: 2, read: 1, pct: 50 })
    expect(progress([]).pct).toBe(100)
  })
  it('срок: просрочен / скоро / в порядке', () => {
    expect(dueState(d({ dueDate: '2026-11-10' }), now)).toBe('overdue')
    expect(dueState(d({ dueDate: '2026-11-16' }), now)).toBe('soon')
    expect(dueState(d({ dueDate: '2026-12-30' }), now)).toBe('ok')
    expect(dueState(d({ dueDate: '2026-11-10', read: true }), now)).toBe('none')
  })
  it('обновлён после прочтения', () => {
    expect(isUpdatedSinceRead(d({ read: true, readAt: '2026-01-01T00:00:00Z', updatedAt: '2026-02-01T00:00:00Z' }))).toBe(true)
  })
})
