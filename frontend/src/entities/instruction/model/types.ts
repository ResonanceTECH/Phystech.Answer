import type { SourceReference } from '@/entities/source-reference'

// Сохранённая инструкция — результат диалога, который пользователь
// отложил для повторного использования. Хранится локально (MVP);
// backend-эндпоинт /saved-instructions появится позже.
export interface SavedInstruction {
  id: string
  title: string
  answer: string
  sources: SourceReference[]
  createdAt: number
}
