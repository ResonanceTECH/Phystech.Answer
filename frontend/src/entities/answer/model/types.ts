import type { AskSource } from '@/entities/source-reference'

export interface AnswerStep {
  text: string
  refs: number[]
}

export interface AnswerConflict {
  description: string
  refs: number[]
}

export interface AnswerTrace {
  mode: string
  reranked?: boolean
  amendments?: string[]
  model?: string
  latencyMs: number
  retrievalMs?: number
  generationMs?: number
  candidates?: number
}

// Ответ ассистента (/api/ask). Структурированные поля заполняет ML-сервис;
// answer — полная markdown-версия для простых клиентов.
export interface AskResponse {
  status?: 'answered' | 'partial' | 'not_found' | 'invalid' | 'error'
  summary?: string
  steps?: AnswerStep[]
  notes?: string[]
  conflicts?: AnswerConflict[]
  missing?: string
  answer: string
  sources: AskSource[]
  trace?: AnswerTrace
  stub?: boolean
}

export type AnswerStatus =
  | 'answered'
  | 'partial'
  | 'clarification_required'
  | 'not_found'
  | 'conflict'
  | 'error'

// Маркер ответа «информации нет» (для старых ответов без поля status).
const NOT_FOUND_MARKERS = ['нет информации, достаточной', 'не найдено подтверждённой информации']

export function deriveAnswerStatus(resp: AskResponse): AnswerStatus {
  if (resp.stub || resp.status === 'error') return 'error'
  switch (resp.status) {
    case 'not_found':
      return 'not_found'
    case 'invalid':
      return 'clarification_required'
    case 'partial':
      return 'partial'
    case 'answered':
      return resp.conflicts && resp.conflicts.length > 0 ? 'conflict' : 'answered'
  }
  const text = resp.answer.toLowerCase()
  if (NOT_FOUND_MARKERS.some((m) => text.includes(m))) return 'not_found'
  return 'answered'
}
