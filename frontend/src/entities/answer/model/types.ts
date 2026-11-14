import type { AskSource } from '@/entities/source-reference'

// Ответ ассистента. Backend сейчас возвращает плоскую структуру
// {answer, sources, stub}; статус выводится из этих полей.
export interface AskResponse {
  answer: string
  sources: AskSource[]
  stub?: boolean
}

export type AnswerStatus =
  | 'answered'
  | 'partial'
  | 'clarification_required'
  | 'not_found'
  | 'conflict'
  | 'error'

// Маркер ответа «информации нет» — совпадает с текстом ML-сервиса.
const NOT_FOUND_MARKER = 'не найдено подтверждённой информации'

export function deriveAnswerStatus(resp: AskResponse): AnswerStatus {
  if (resp.stub) return 'error'
  if (resp.answer.toLowerCase().includes(NOT_FOUND_MARKER)) return 'not_found'
  return 'answered'
}
