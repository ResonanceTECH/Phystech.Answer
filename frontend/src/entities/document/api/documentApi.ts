import { api } from '@/shared/api'
import type { Document, DocumentInput } from '../model/types'

export function listDocuments(userId?: string): Promise<Document[]> {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return api.get<Document[]>(`/v1/documents${qs}`)
}

/** Каталог без полных текстов; статус прочтения — для текущего пользователя. */
export function listDocumentsBrief(): Promise<Document[]> {
  return api.get<Document[]>('/v1/documents?brief=1')
}

export function markDocumentRead(id: string): Promise<{ readAt: string }> {
  return api.post<{ readAt: string }>(`/v1/documents/${id}/read`)
}

export function getDocument(id: string): Promise<Document> {
  return api.get<Document>(`/v1/documents/${id}`)
}

export function createDocument(input: DocumentInput): Promise<Document> {
  return api.post<Document>('/v1/documents', input)
}

export function updateDocument(
  id: string,
  input: DocumentInput,
): Promise<Document> {
  return api.put<Document>(`/v1/documents/${id}`, input)
}

export function deleteDocument(id: string): Promise<void> {
  return api.delete<void>(`/v1/documents/${id}`)
}
