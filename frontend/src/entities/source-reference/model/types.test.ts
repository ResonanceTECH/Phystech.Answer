import { describe, expect, it } from 'vitest'
import { formatDocDate, toSourceReference } from './types'
import type { AskSource } from './types'

const src: AskSource = {
  ref: 2,
  documentId: 'doc-1',
  title: 'Регламент командировок',
  number: '794-1',
  date: '2023-03-28',
  section: 'п. 3.1–3.4',
  heading: '3. Согласование',
  excerpt: 'Работник обязан сдать авансовый отчёт…',
  score: 0.87,
}

describe('toSourceReference', () => {
  it('маппит поля ML-формата в сущность', () => {
    expect(toSourceReference(src, 0)).toEqual({
      id: 'doc-1#2',
      ref: 2,
      documentId: 'doc-1',
      documentTitle: 'Регламент командировок',
      documentNumber: '794-1',
      documentDate: '2023-03-28',
      sectionNumber: '3.1–3.4',
      heading: '3. Согласование',
      quote: 'Работник обязан сдать авансовый отчёт…',
      score: 0.87,
    })
  })

  it('без ref нумерует по порядку, пустая секция → undefined', () => {
    const r = toSourceReference({ ...src, ref: undefined, section: '' }, 1)
    expect(r.ref).toBe(2)
    expect(r.sectionNumber).toBeUndefined()
  })
})

describe('formatDocDate', () => {
  it('ISO → ДД.ММ.ГГГГ', () => {
    expect(formatDocDate('2023-03-28')).toBe('28.03.2023')
    expect(formatDocDate(undefined)).toBe('')
  })
})
