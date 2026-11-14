import { api } from '@/shared/api'
import type { Document, DocumentInput } from '../model/types'

export function listDocuments(userId?: string): Promise<Document[]> {
  const qs = userId ? `?userId=${encodeURIComponent(userId)}` : ''
  return api.get<Document[]>(`/documents${qs}`)
}

export function getDocument(id: string): Promise<Document> {
  return api.get<Document>(`/documents/${id}`)
}

export function createDocument(input: DocumentInput): Promise<Document> {
  return api.post<Document>('/documents', input)
}

export function updateDocument(
  id: string,
  input: DocumentInput,
): Promise<Document> {
  return api.put<Document>(`/documents/${id}`, input)
}

export function deleteDocument(id: string): Promise<void> {
  return api.delete<void>(`/documents/${id}`)
}
