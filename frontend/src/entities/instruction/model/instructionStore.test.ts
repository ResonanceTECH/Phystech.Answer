import { beforeEach, describe, expect, it } from 'vitest'
import { useInstructionStore } from './instructionStore'

describe('instructionStore', () => {
  beforeEach(() => {
    useInstructionStore.setState({ instructions: [] })
  })

  it('save создаёт инструкцию с id и createdAt', () => {
    const id = useInstructionStore.getState().save({
      title: 'Как уволиться',
      answer: 'Шаг 1…',
      sources: [],
    })
    const ins = useInstructionStore.getState().instructions[0]
    expect(ins.id).toBe(id)
    expect(ins.title).toBe('Как уволиться')
    expect(ins.createdAt).toBeGreaterThan(0)
  })

  it('remove удаляет инструкцию', () => {
    const store = useInstructionStore.getState()
    const id = store.save({ title: 'a', answer: 'b', sources: [] })
    store.remove(id)
    expect(useInstructionStore.getState().instructions).toHaveLength(0)
  })
})
