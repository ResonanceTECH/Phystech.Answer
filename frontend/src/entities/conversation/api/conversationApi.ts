import { api } from '@/shared/api'
import type { SourceReference } from '@/entities/source-reference'
import type { AnswerConflict, AnswerStatus, AnswerStep } from '@/entities/answer'

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
    ref: i + 1, // [n] markers in steps point at the n-th citation
    documentId: c.document_id ?? '',
    documentTitle: c.document_title,
    sectionNumber: c.section_number || undefined,
    quote: c.quote,
    score: c.score,
  }
}

const CONFLICT_PREFIX = 'Конфликт норм: '
const REF_RE = /\[(\d+)\]/g

function refsOf(text: string): number[] {
  return [...new Set([...text.matchAll(REF_RE)].map((m) => Number(m[1])))]
}

// Раскладывает сохранённый ответ run'а по полям карточки ответа:
// шаги с [n]-ссылками, противоречия, примечания и «чего не хватает».
export function toAnswerFields(ans: ServerAnswer): {
  summary: string
  steps: AnswerStep[]
  conflicts: AnswerConflict[]
  notes: string[]
  missing?: string
} {
  const steps = [...(ans.steps ?? [])]
    .sort((a, b) => a.order - b.order)
    .map((st) => {
      const text = st.description || st.title
      return { text, refs: refsOf(text) }
    })
  const conflicts: AnswerConflict[] = []
  const notes: string[] = []
  for (const w of ans.warnings ?? []) {
    if (w.startsWith(CONFLICT_PREFIX)) {
      const description = w.slice(CONFLICT_PREFIX.length)
      conflicts.push({ description, refs: refsOf(description) })
    } else {
      notes.push(w)
    }
  }
  const missing = (ans.missing_information ?? []).join(' ').trim()
  return { summary: ans.summary, steps, conflicts, notes, missing: missing || undefined }
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
