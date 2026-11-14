import type { SourceReference } from '@/entities/source-reference'
import type { AnswerConflict, AnswerStatus, AnswerStep, AnswerTrace } from '@/entities/answer'

export interface Conversation {
  id: string
  title: string
  createdAt: number
  updatedAt: number
}

export interface ChatMessage {
  id: string
  conversationId: string
  role: 'user' | 'assistant'
  text: string
  sources?: SourceReference[]
  status?: AnswerStatus
  summary?: string
  steps?: AnswerStep[]
  notes?: string[]
  conflicts?: AnswerConflict[]
  missing?: string
  trace?: AnswerTrace
  stub?: boolean
  pending?: boolean
  // Серверные идентификаторы для «Сохранить инструкцию»/маршрутов.
  answerId?: string
  runId?: string
  warnings?: string[]
  missingInformation?: string[]
  createdAt: number
}
