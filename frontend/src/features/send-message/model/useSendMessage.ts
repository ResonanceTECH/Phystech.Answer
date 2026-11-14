import { useState } from 'react'
import {
  conversationApi,
  toAnswerFields,
  toSourceReference,
  type ServerAnswer,
} from '@/entities/conversation'
import { useConversationStore } from '@/entities/conversation'
import type { ChatMessage } from '@/entities/conversation'
import { apiErrorMessage } from '@/shared/api'

let seq = 0
function messageId(prefix: string) {
  seq += 1
  return `${prefix}_${Date.now().toString(36)}_${seq}`
}

const POLL_INTERVAL_MS = 900
const RUN_TIMEOUT_MS = 180_000

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

function mapAnswer(convId: string, pendingId: string, runId: string, ans: ServerAnswer) {
  const fields = toAnswerFields(ans)
  return {
    conversationId: convId,
    messageId: pendingId,
    patch: {
      text: ans.summary,
      summary: fields.summary,
      steps: fields.steps,
      conflicts: fields.conflicts,
      notes: fields.notes,
      missing: fields.missing,
      sources: (ans.sources ?? []).map(toSourceReference),
      status: ans.status,
      stub: false,
      pending: false,
      answerId: ans.answer_id,
      runId,
      warnings: ans.warnings ?? [],
      missingInformation: ans.missing_information ?? [],
    } satisfies Partial<ChatMessage>,
  }
}

// Ожидание завершения run: backend кладёт answer.generate в очередь,
// worker обрабатывает асинхронно — опрашиваем статус до терминального.
async function waitForRun(runId: string): Promise<'completed' | 'failed'> {
  const deadline = Date.now() + RUN_TIMEOUT_MS
  for (;;) {
    const run = await conversationApi.getRun(runId)
    if (run.status === 'completed') return 'completed'
    if (run.status === 'failed' || run.status === 'cancelled') return 'failed'
    if (Date.now() > deadline) return 'failed'
    await sleep(POLL_INTERVAL_MS)
  }
}

// Сценарий отправки: user-сообщение → POST /conversations/{id}/messages
// → run в очереди → poll → GET /runs/{id}/answer → финальный ответ
// с источниками, статусом и answer_id (для «Сохранить инструкцию»).
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
      const { run_id } = await conversationApi.postMessage(conversationId, text)
      const outcome = await waitForRun(run_id)
      if (outcome !== 'completed') {
        throw new Error('answer run did not complete')
      }
      const ans = await conversationApi.getRunAnswer(run_id)
      const { patch } = mapAnswer(conversationId, pendingId, run_id, ans)
      updateMessage(conversationId, pendingId, patch)
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
