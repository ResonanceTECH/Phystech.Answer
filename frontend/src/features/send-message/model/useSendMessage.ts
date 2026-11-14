import { useState } from 'react'
import { useConversationStore } from '@/entities/conversation'
import { deriveAnswerStatus } from '@/entities/answer'
import { toSourceReference } from '@/entities/source-reference'
import { apiErrorMessage } from '@/shared/api'
import { sendMessage } from '../api/sendMessage'

let seq = 0
function messageId(prefix: string) {
  seq += 1
  return `${prefix}_${Date.now().toString(36)}_${seq}`
}

// Сценарий отправки сообщения: user-сообщение → pending assistant →
// ответ со статусом и источниками (или ошибка).
// Контекст диалога хранит backend; для MVP история сообщений —
// клиентская (entities/conversation), запрос содержит только вопрос.
export function useSendMessage() {
  const appendMessage = useConversationStore((s) => s.appendMessage)
  const updateMessage = useConversationStore((s) => s.updateMessage)
  const [isPending, setIsPending] = useState(false)

  const send = async (conversationId: string, question: string) => {
    const text = question.trim()
    if (!text || isPending) return

    setIsPending(true)
    const now = Date.now()
    appendMessage(conversationId, {
      id: messageId('u'),
      conversationId,
      role: 'user',
      text,
      createdAt: now,
    })
    const pendingId = messageId('a')
    appendMessage(conversationId, {
      id: pendingId,
      conversationId,
      role: 'assistant',
      text: '',
      pending: true,
      createdAt: now,
    })

    try {
      const resp = await sendMessage(text)
      updateMessage(conversationId, pendingId, {
        text: resp.answer,
        sources: (resp.sources ?? []).map(toSourceReference),
        status: deriveAnswerStatus(resp),
        summary: resp.summary,
        steps: resp.steps,
        notes: resp.notes,
        conflicts: resp.conflicts,
        missing: resp.missing,
        trace: resp.trace,
        stub: resp.stub ?? false,
        pending: false,
      })
    } catch (err) {
      updateMessage(conversationId, pendingId, {
        text: `Не удалось получить ответ: ${apiErrorMessage(err)}`,
        status: 'error',
        pending: false,
      })
    } finally {
      setIsPending(false)
    }
  }

  return { send, isPending }
}
