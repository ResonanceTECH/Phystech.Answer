import type { SourceReference } from '@/entities/source-reference'
import type { AnswerStatus } from '@/entities/answer'

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
  stub?: boolean
  pending?: boolean
  // Серверные идентификаторы для «Сохранить инструкцию»/маршрутов.
  answerId?: string
  runId?: string
  warnings?: string[]
  missingInformation?: string[]
  createdAt: number
}
