// Оценка надёжности пароля 0–4 без внешних зависимостей.
// Политика MVP: длина важнее набора символов (спека §4.2).
export function passwordScore(pw: string): 0 | 1 | 2 | 3 | 4 {
  if (!pw) return 0
  // Короткий пароль слаб независимо от набора символов.
  if (pw.length < 8) return 1
  let score = 1
  if (pw.length >= 12) score++
  if (pw.length >= 16) score++
  const variety =
    Number(/[a-zа-яё]/.test(pw) && /[A-ZА-ЯЁ]/.test(pw)) +
    Number(/\d/.test(pw)) +
    Number(/[^\w\sа-яА-ЯёЁ]/.test(pw))
  score = Math.min(4, score + Math.max(0, variety - 1))
  return score as 0 | 1 | 2 | 3 | 4
}

export const passwordScoreLabels = [
  'Очень слабый',
  'Слабый',
  'Средний',
  'Хороший',
  'Надёжный',
] as const
