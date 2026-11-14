// Сущность SourceReference — ссылка на положение документа,
// подтверждающее часть ответа ассистента.

export interface SourceReference {
  id: string
  /** Номер [n], которым ответ ссылается на источник. */
  ref: number
  documentId: string
  documentTitle: string
  documentNumber?: string
  documentDate?: string
  sectionNumber?: string
  heading?: string
  quote: string
  score?: number
}

// AskSource — формат источника, который возвращает /api/ask.
export interface AskSource {
  ref?: number
  documentId: string
  title: string
  number?: string
  date?: string
  section?: string
  heading?: string
  excerpt: string
  score: number
}

export function toSourceReference(src: AskSource, index: number): SourceReference {
  return {
    id: `${src.documentId}#${src.ref ?? index + 1}`,
    ref: src.ref ?? index + 1,
    documentId: src.documentId,
    documentTitle: src.title,
    documentNumber: src.number || undefined,
    documentDate: src.date || undefined,
    // ML возвращает метку вида «п. 2.1–2.5»; храним без префикса.
    sectionNumber: src.section ? src.section.replace(/^п\.\s*/, '') : undefined,
    heading: src.heading || undefined,
    quote: src.excerpt,
    score: src.score,
  }
}

/** 2023-03-28 → 28.03.2023 */
export function formatDocDate(date?: string): string {
  if (!date) return ''
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(date)
  return m ? `${m[3]}.${m[2]}.${m[1]}` : date
}
