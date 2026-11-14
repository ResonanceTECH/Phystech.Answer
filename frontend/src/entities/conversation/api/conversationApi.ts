import { api } from '@/shared/api'
import type { SourceReference } from '@/entities/source-reference'
import type { AnswerStatus } from '@/entities/answer'

// Типы ответов backend /api/v1 (conversations/runs/answers/instructions/routes).

export interface ServerConversation {
  id: string
  userId?: string
  title: string
  createdAt: string
  updatedAt: string
}

export interface ServerMessage {
  id: string
  conversationId: string
  role: 'user' | 'assistant'
  content: string
  createdAt: string
}

export interface ServerRun {
  id: string
  conversationId: string
  status: 'queued' | 'running' | 'completed' | 'failed' | 'cancelled'
  errorCode?: string
}

export interface ServerStep {
  id: string
  order: number
  title: string
  description: string
  responsible?: string | null
  deadline?: string | null
  condition?: string | null
  required: boolean
}

export interface ServerCitation {
  id: string
  document_id?: string
  document_title: string
  section_number?: string
  page_number?: number | null
  quote: string
  score: number
}

export interface ServerAnswer {
  answer_id: string
  conversation_id?: string
  status: AnswerStatus
  summary: string
  steps: ServerStep[]
  sources: ServerCitation[]
  warnings: string[]
  missing_information: string[]
}

export interface ServerSavedInstruction {
  id: string
  userId: string
  answerId: string
  title: string
  createdAt: string
}

export interface ServerActionRouteStep {
  stepId: string
  order: number
  title: string
  done?: boolean
}

export interface ServerActionRoute {
  id: string
  userId: string
  answerId: string
  title: string
  stale: boolean
  createdAt: string
  updatedAt: string
}

export function toSourceReference(c: ServerCitation, i: number): SourceReference {
  return {
    id: c.id || `cit_${i}`,
    documentId: c.document_id ?? '',
    documentTitle: c.document_title,
    sectionNumber: c.section_number || undefined,
    quote: c.quote,
    score: c.score,
  }
}

export const conversationApi = {
  list: () => api.get<ServerConversation[]>('/conversations'),

  create: (input: { title?: string; message?: string }) =>
    api.post<
      | ServerConversation
      | { conversation: ServerConversation; message_id: string; run_id: string; status: string }
    >('/conversations', input),

  get: (id: string) =>
    api.get<{ conversation: ServerConversation; messages: ServerMessage[] }>(
      `/conversations/${id}`,
    ),

  rename: (id: string, title: string) =>
    api.patch<ServerConversation>(`/conversations/${id}`, { title }),

  remove: (id: string) => api.delete<void>(`/conversations/${id}`),

  postMessage: (id: string, content: string) =>
    api.post<{ message_id: string; run_id: string; status: string }>(
      `/conversations/${id}/messages`,
      { content },
    ),

  getRun: (id: string) => api.get<ServerRun>(`/runs/${id}`),

  getRunAnswer: (id: string) => api.get<ServerAnswer>(`/runs/${id}/answer`),

  getAnswer: (id: string) => api.get<ServerAnswer>(`/answers/${id}`),
};

export const savedInstructionApi = {
  list: () => api.get<ServerSavedInstruction[]>('/saved-instructions'),
  create: (answerId: string, title: string) =>
    api.post<ServerSavedInstruction>('/saved-instructions', { answerId, title }),
  remove: (id: string) => api.delete<void>(`/saved-instructions/${id}`),
};

export const actionRouteApi = {
  list: () => api.get<ServerActionRoute[]>('/action-routes'),
  create: (answerId: string, title: string) =>
    api.post<ServerActionRoute>('/action-routes', { answerId, title }),
  get: (id: string) =>
    api.get<{ route: ServerActionRoute; steps: ServerActionRouteStep[] }>(
      `/action-routes/${id}`,
    ),
  setStepDone: (routeId: string, stepId: string, done: boolean) =>
    api.patch<{ status: string }>(`/action-routes/${routeId}/steps/${stepId}`, { done }),
};
