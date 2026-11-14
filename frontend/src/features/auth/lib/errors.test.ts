import { describe, expect, it } from 'vitest'
import { ApiError } from '@/shared/api/errors'
import { authErrorMessage } from './errors'

const apiErr = (status: number, code: string) =>
  new ApiError(status, 'msg', { error: { code } })

describe('authErrorMessage', () => {
  it('маппит коды API в конкретные причины', () => {
    expect(authErrorMessage(apiErr(401, 'INVALID_CREDENTIALS'))).toContain('почту и пароль')
    expect(authErrorMessage(apiErr(409, 'EMAIL_ALREADY_EXISTS'))).toContain('уже существует')
    expect(authErrorMessage(apiErr(429, 'RATE_LIMITED'))).toContain('много попыток')
    expect(authErrorMessage(apiErr(400, 'RESET_TOKEN_EXPIRED'))).toContain('истёк')
  })

  it('неизвестный код → сообщение по статусу', () => {
    expect(authErrorMessage(apiErr(500, 'INTERNAL'))).toBe('Внутренняя ошибка сервера.')
  })

  it('сетевая ошибка → «сервер недоступен»', () => {
    expect(authErrorMessage(new TypeError('fetch failed'))).toContain('сервером')
  })
})
