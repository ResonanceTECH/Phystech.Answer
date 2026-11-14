import { describe, expect, it } from 'vitest'
import { buildMatcher, queryStems, snippets, splitByMatches } from './highlight'

describe('queryStems', () => {
  it('сводит слова к основам, номера оставляет точно, стоп-слова убирает', () => {
    expect(queryStems('Как оформить командировку по приказу 794-1')).toEqual(['командиро', 'оформ', 'прика', '794-1'])
  })
})

describe('splitByMatches', () => {
  it('подсвечивает разные формы слова, учитывая ё/е и регистр', () => {
    const segs = splitByMatches('Командировки и командированию; счёт ФЛС', buildMatcher('командировка счет'))
    expect(segs.filter((s) => s.hit).map((s) => s.text)).toEqual(['Командировки', 'командированию', 'счёт'])
  })
  it('не подсвечивает середину слова', () => {
    const segs = splitByMatches('перекомандировка', buildMatcher('командировка'))
    expect(segs.some((s) => s.hit)).toBe(false)
  })
})

describe('snippets', () => {
  it('возвращает фрагмент вокруг совпадения и общее число совпадений', () => {
    const text = 'Начало. '.repeat(30) + 'Служебная записка подаётся за два месяца. ' + 'Конец. '.repeat(30) + 'Ещё одна записка.'
    const sn = snippets(text, 'служебную записку', 2, 40)
    expect(sn?.total).toBe(3)
    expect(sn?.segments.some((s) => s.hit && s.text === 'Служебная')).toBe(true)
  })
  it('null, если совпадений нет', () => {
    expect(snippets('текст', 'отпуск')).toBeNull()
  })
})
