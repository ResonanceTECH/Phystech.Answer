import { Box, Typography } from '@mui/material'
import { useEffect, useRef } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import { useConversationStore } from '@/entities/conversation'
import { useSendMessage } from '@/features/send-message'
import { paths } from '@/app/router/routePaths'
import { ChatWorkspace } from '@/widgets/chat-workspace'
import { EmptyState } from '@/shared/ui'
import type { ChatMessage } from '@/entities/conversation'

const NO_MESSAGES: ChatMessage[] = []

// Диалог с ассистентом. /app/chat/new — новый диалог (создаётся при
// первом сообщении), /app/chat/:conversationId — существующий.
export function ChatPage() {
  const { conversationId: paramId } = useParams<{ conversationId: string }>()
  const location = useLocation()
  const navigate = useNavigate()
  const { send, isPending } = useSendMessage()

  const isNew = !paramId || paramId === 'new'
  const conversationId = isNew ? null : paramId

  const createConversation = useConversationStore((s) => s.createConversation)
  const conversations = useConversationStore((s) => s.conversations)
  const messages = useConversationStore((s) =>
    conversationId ? (s.messages[conversationId] ?? NO_MESSAGES) : NO_MESSAGES,
  )

  const activeConversation = conversations.find((c) => c.id === conversationId)
  const autoSentRef = useRef(false)

  const handleSend = async (text: string) => {
    let convId = conversationId
    if (!convId) {
      convId = createConversation(text)
      navigate(paths.app.chat(convId), { replace: true })
    }
    await send(convId, text)
  }

  // Авто-отправка вопроса, переданного через router state (главная,
  // карточка документа).
  useEffect(() => {
    const state = location.state as { question?: string } | null
    if (state?.question && !autoSentRef.current) {
      autoSentRef.current = true
      void handleSend(state.question)
      navigate(location.pathname, { replace: true, state: {} })
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.state])

  // Несуществующий диалог (например, удалённый).
  if (!isNew && !activeConversation && conversations.length > 0) {
    return (
      <EmptyState
        title="Диалог не найден"
        description="Возможно, он был удалён. Начните новый диалог."
        action={
          <Typography variant="body2">
            <a href={paths.app.chatNew}>Новый диалог</a>
          </Typography>
        }
      />
    )
  }

  return (
    <Box sx={{ height: '100%', minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <ChatWorkspace
        messages={messages}
        conversationTitle={activeConversation?.title}
        disabled={isPending}
        onSend={(text) => void handleSend(text)}
      />
    </Box>
  )
}
