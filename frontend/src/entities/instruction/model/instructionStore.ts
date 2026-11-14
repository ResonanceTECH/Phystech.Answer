import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import type { SavedInstruction } from './types'

interface InstructionState {
  instructions: SavedInstruction[]
  save: (instruction: Omit<SavedInstruction, 'id' | 'createdAt'>) => string
  remove: (id: string) => void
}

export const useInstructionStore = create<InstructionState>()(
  persist(
    (set) => ({
      instructions: [],

      save: (instruction) => {
        const id = `ins_${Date.now().toString(36)}_${Math.random().toString(36).slice(2, 8)}`
        set((s) => ({
          instructions: [
            { ...instruction, id, createdAt: Date.now() },
            ...s.instructions,
          ],
        }))
        return id
      },

      remove: (id) =>
        set((s) => ({
          instructions: s.instructions.filter((i) => i.id !== id),
        })),
    }),
    { name: 'phystech.instructions.v1' },
  ),
)
