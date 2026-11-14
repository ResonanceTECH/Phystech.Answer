// Сущность Document — нормативный документ базы знаний.

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

export interface DocumentInput {
  title: string
  content: string
  tags?: string[]
}

export interface DocumentListItem extends Document {}
