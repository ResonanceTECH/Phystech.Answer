import { describe, expect, it } from 'vitest'
import { passwordScore } from './passwordStrength'

describe('passwordScore', () => {
  it('пустой пароль — 0', () => {
    expect(passwordScore('')).toBe(0)
  })

  it('короткий пароль — слабый', () => {
    expect(passwordScore('abc')).toBe(1)
    expect(passwordScore('password')).toBeLessThanOrEqual(2)
  })

  it('длина важнее набора символов', () => {
    const weak = 'Aa1!'
    const long = 'это-очень-длинный-пароль-без-цифр'
    expect(passwordScore(long)).toBeGreaterThan(passwordScore(weak))
  })

  it('длинный разнообразный пароль — максимум', () => {
    expect(passwordScore('Str0ng!Password#2026_xY')).toBe(4)
  })

  it('счёт в диапазоне 0–4', () => {
    for (const pw of ['a', 'aaaaaaaaaaaaaa', 'A1!aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa']) {
      const s = passwordScore(pw)
      expect(s).toBeGreaterThanOrEqual(0)
      expect(s).toBeLessThanOrEqual(4)
    }
  })
})
