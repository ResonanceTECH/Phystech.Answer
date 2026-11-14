export interface Document {
  id: string
  title: string
  content: string
  tags: string[]
  version: number
  createdAt: string
  updatedAt: string
  read?: boolean
  readAt?: string
}

export interface AskSource {
  documentId: string
  title: string
  section?: string
  excerpt: string
  score: number
}

export interface AskResponse {
  answer: string
  sources: AskSource[]
  stub?: boolean
}
