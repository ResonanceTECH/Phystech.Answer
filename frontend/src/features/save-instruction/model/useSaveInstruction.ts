import { useCallback } from 'react'
import { useInstructionStore } from '@/entities/instruction'
import type { SavedInstruction } from '@/entities/instruction'

// Действие «Сохранить инструкцию»: складывает ответ ассистента
// в локальное хранилище сохранённых инструкций.
export function useSaveInstruction() {
  const save = useInstructionStore((s) => s.save)
  return useCallback(
    (input: Omit<SavedInstruction, 'id' | 'createdAt'>) => save(input),
    [save],
  )
}
