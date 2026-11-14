import { describe, expect, it } from 'vitest'
import { toSourceReference } from './types'
import type { AskSource } from './types'

const src: AskSource = {
  documentId: 'doc-1',
  title: 'Регламент командировок',
  section: '3.2',
  excerpt: 'Работник обязан сдать авансовый отчёт…',
  score: 0.87,
}

describe('toSourceReference', () => {
  it('маппит поля backend-формата в сущность', () => {
    const ref = toSourceReference(src, 0)
    expect(ref).toEqual({
      id: 'doc-1#0',
      documentId: 'doc-1',
      documentTitle: 'Регламент командировок',
      sectionNumber: '3.2',
      quote: 'Работник обязан сдать авансовый отчёт…',
      score: 0.87,
    })
  })

  it('пустая секция → undefined', () => {
    expect(toSourceReference({ ...src, section: '' }, 1).sectionNumber).toBeUndefined()
  })
})
