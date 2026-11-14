import { describe, expect, it } from 'vitest'
import { deriveAnswerStatus } from './types'
import type { AskResponse } from './types'

const base: AskResponse = { answer: 'Сделайте X.', sources: [] }

describe('deriveAnswerStatus', () => {
  it('answered для обычного ответа', () => {
    expect(deriveAnswerStatus(base)).toBe('answered')
  })

  it('not_found по маркеру «не найдено подтверждённой информации»', () => {
    expect(
      deriveAnswerStatus({
        ...base,
        answer: 'По вашему запросу не найдено подтверждённой информации в базе.',
      }),
    ).toBe('not_found')
  })

  it('error для stub-ответа ML-фолбэка', () => {
    expect(deriveAnswerStatus({ ...base, stub: true })).toBe('error')
  })
})
