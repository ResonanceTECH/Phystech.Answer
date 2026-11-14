import { describe, expect, it } from 'vitest'
import { ApiError, apiErrorMessage } from './errors'

describe('apiErrorMessage', () => {
  it('маппит HTTP-статусы в понятные сообщения', () => {
    expect(apiErrorMessage(new ApiError(404, 'not found'))).toBe('Ресурс не найден.')
    expect(apiErrorMessage(new ApiError(403, 'forbidden'))).toBe('Недостаточно прав.')
    expect(apiErrorMessage(new ApiError(500, 'oops'))).toBe('Внутренняя ошибка сервера.')
  })

  it('обычная Error → её message', () => {
    expect(apiErrorMessage(new Error('сеть недоступна'))).toBe('сеть недоступна')
  })

  it('неизвестное → fallback', () => {
    expect(apiErrorMessage(null)).toBe('Неизвестная ошибка.')
    expect(apiErrorMessage('строка')).toBe('Неизвестная ошибка.')
  })
})
