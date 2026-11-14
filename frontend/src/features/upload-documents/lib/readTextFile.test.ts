import { describe, expect, it } from 'vitest'
import {
  isSupportedTextFile,
  titleFromFileName,
} from './readTextFile'

describe('isSupportedTextFile', () => {
  it('принимает txt/md', () => {
    expect(isSupportedTextFile(new File(['x'], 'reglament.txt'))).toBe(true)
    expect(isSupportedTextFile(new File(['x'], 'REG.MD'))).toBe(true)
  })

  it('отклоняет pdf/docx', () => {
    expect(isSupportedTextFile(new File(['x'], 'doc.pdf'))).toBe(false)
    expect(isSupportedTextFile(new File(['x'], 'doc.docx'))).toBe(false)
  })
})

describe('titleFromFileName', () => {
  it('убирает расширение', () => {
    expect(titleFromFileName('Порядок отпуска.txt')).toBe('Порядок отпуска')
    expect(titleFromFileName('a.b.md')).toBe('a.b')
  })
})
