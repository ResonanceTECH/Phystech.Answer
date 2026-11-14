// Сущность SourceReference — ссылка на положение документа,
// подтверждающее часть ответа ассистента.

export interface SourceReference {
  id: string
  documentId: string
  documentTitle: string
  sectionNumber?: string
  quote: string
  score?: number
}

// AskSource — формат, который возвращает backend /api/ask.
export interface AskSource {
  documentId: string
  title: string
  section?: string
  excerpt: string
  score: number
}

export function toSourceReference(src: AskSource, index: number): SourceReference {
  return {
    id: `${src.documentId}#${index}`,
    documentId: src.documentId,
    documentTitle: src.title,
    sectionNumber: src.section || undefined,
    quote: src.excerpt,
    score: src.score,
  }
}
