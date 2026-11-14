import { api } from '@/shared/api'
import type { AskResponse } from '@/entities/answer'

// POST /api/ask — вопрос к базе регламентов через backend-прокси к ML.
export function sendMessage(question: string): Promise<AskResponse> {
  return api.post<AskResponse>('/ask', { question })
}
