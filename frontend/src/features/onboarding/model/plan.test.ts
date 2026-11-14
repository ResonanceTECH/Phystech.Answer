import { describe, expect, it } from 'vitest'
import type { Document } from '@/entities/document'
import { buildPlan, isNewDocument, isUpdatedSinceRead } from './plan'

const doc = (p: Partial<Document>): Document => ({
  id: p.title ?? 'x',
  title: 'Документ',
  content: '',
  tags: [],
  version: 1,
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:00Z',
  ...p,
})

describe('buildPlan', () => {
  const docs = [
    doc({ title: 'Приказ и Регламент работы первой линии техподдержки', read: true }),
    doc({ title: 'Регламент о командировании' }),
    doc({ title: 'Приказ N 794-1 от 28.03.2023', documentNumber: '794-1' }),
    doc({ title: 'Приложение 1а Бланк сметы', documentType: 'приложение' }),
    doc({ title: 'Положение о чём-то ещё' }),
  ]
  const plan = buildPlan(docs)

  it('раскладывает документы по модулям, формы — в справочные', () => {
    expect(plan.modules.map((m) => m.id)).toEqual(['workplace', 'people', 'projects'])
    expect(plan.reference).toHaveLength(1)
    expect(plan.extra).toHaveLength(1)
  })

  it('считает прогресс без справочных материалов', () => {
    expect(plan.total).toBe(4)
    expect(plan.read).toBe(1)
    expect(plan.next?.title).toBe('Регламент о командировании')
  })
})

describe('бейджи', () => {
  const now = Date.parse('2026-11-14T00:00:00Z')
  it('новый — за последние 90 дней', () => {
    expect(isNewDocument(doc({ documentDate: '2026-11-03' }), now)).toBe(true)
    expect(isNewDocument(doc({ documentDate: '2025-09-01' }), now)).toBe(false)
  })
  it('обновлён после прочтения', () => {
    expect(isUpdatedSinceRead(doc({ read: true, readAt: '2026-01-01T00:00:00Z', updatedAt: '2026-02-01T00:00:00Z' }))).toBe(true)
    expect(isUpdatedSinceRead(doc({ read: true, readAt: '2026-03-01T00:00:00Z', updatedAt: '2026-02-01T00:00:00Z' }))).toBe(false)
  })
})
