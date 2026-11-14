import { beforeEach, describe, expect, it } from 'vitest'
import { useConversationStore } from './conversationStore'
import type { ChatMessage } from './types'

function reset() {
  useConversationStore.setState({ conversations: [], messages: {} })
}

function msg(id: string, conversationId: string): ChatMessage {
  return {
    id,
    conversationId,
    role: 'user',
    text: 'вопрос',
    createdAt: Date.now(),
  }
}

describe('conversationStore', () => {
  beforeEach(reset)

  it('createConversation добавляет диалог и возвращает id', () => {
    const id = useConversationStore.getState().createConversation('Как уволиться?')
    const conv = useConversationStore.getState().conversations[0]
    expect(conv.id).toBe(id)
    expect(conv.title).toBe('Как уволиться?')
  })

  it('appendMessage + updateMessage обновляют pending-сообщение', () => {
    const store = useConversationStore.getState()
    const convId = store.createConversation('тема')
    store.appendMessage(convId, msg('m1', convId))
    store.appendMessage(convId, { ...msg('m2', convId), role: 'assistant', pending: true })

    store.updateMessage(convId, 'm2', { text: 'ответ', pending: false })

    const messages = useConversationStore.getState().messages[convId]
    expect(messages).toHaveLength(2)
    expect(messages[1]).toMatchObject({ text: 'ответ', pending: false })
  })

  it('deleteConversation удаляет диалог и его сообщения', () => {
    const store = useConversationStore.getState()
    const convId = store.createConversation('тема')
    store.appendMessage(convId, msg('m1', convId))
    store.deleteConversation(convId)

    const s = useConversationStore.getState()
    expect(s.conversations).toHaveLength(0)
    expect(s.messages[convId]).toBeUndefined()
  })

  it('deriveTitle обрезает длинные заголовки', () => {
    const long = 'очень '.repeat(30)
    const id = useConversationStore.getState().createConversation(long)
    const title = useConversationStore.getState().conversations[0].title
    expect(title.length).toBeLessThanOrEqual(60)
    expect(title.endsWith('…')).toBe(true)
    expect(id).toBeTruthy()
  })
})
