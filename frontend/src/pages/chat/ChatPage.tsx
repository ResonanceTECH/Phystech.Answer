import { Box, Typography } from '@mui/material'
import { useEffect, useRef } from 'react'
import { useLocation, useNavigate, useParams } from 'react-router-dom'
import {
  conversationApi,
  useConversationStore,
  type ChatMessage,
} from '@/entities/conversation'
import { useSendMessage } from '@/features/send-message'
import { paths } from '@/app/router/routePaths'
import { ChatWorkspace } from '@/widgets/chat-workspace'
import { EmptyState } from '@/shared/ui'

const NO_MESSAGES: ChatMessage[] = []

function deriveTitle(text: string) {
  const clean = text.replace(/\s+/g, ' ').trim()
  return clean.length > 60 ? `${clean.slice(0, 57)}…` : clean || 'Новый диалог'
}

// Диалог с ассистентом. /app/chat/new — новый диалог (создаётся на сервере
// при первом сообщении), /app/chat/:conversationId — существующий.
// История и ответы — серверные (PostgreSQL через backend API).
export function ChatPage() {
  const { conversationId: paramId } = useParams<{ conversationId: string }>()
  const location = useLocation()
  const navigate = useNavigate()
  const { send, isPending } = useSendMessage()

  const isNew = !paramId || paramId === 'new'
  const conversationId = isNew ? null : paramId

  const upsertConversation = useConversationStore((s) => s.upsertConversation)
  const setMessages = useConversationStore((s) => s.setMessages)
  const conversations = useConversationStore((s) => s.conversations)
  const messages = useConversationStore((s) =>
    conversationId ? (s.messages[conversationId] ?? NO_MESSAGES) : NO_MESSAGES,
  )

  const activeConversation = conversations.find((c) => c.id === conversationId)
  const autoSentRef = useRef(false)

  // Гидратация сообщений диалога с сервера при открытии.
  useEffect(() => {
    if (!conversationId) return
    let cancelled = false
    conversationApi
      .get(conversationId)
      .then(({ messages: serverMsgs }) => {
        if (cancelled) return
        // Не затираем локальные сообщения живой сессии (в них есть
        // sources/status/answerId, которых нет в текстовой выдаче сервера).
        const local = useConversationStore.getState().messages[conversationId]
        if (local && local.length > 0) return
        setMessages(
          conversationId,
          serverMsgs.map((m) => ({
            id: m.id,
            conversationId: m.conversationId,
            role: m.role,
            text: m.content,
            createdAt: Date.parse(m.createdAt),
          })),
        )
      })
      .catch(() => {
        // Диалог мог быть удалён — покажем empty state ниже.
      })
    return () => {
      cancelled = true
    }
  }, [conversationId, setMessages])

  const handleSend = async (text: string) => {
    const question = text.trim()
    if (!question) return

    let convId = conversationId
    if (!convId) {
      // Создаём диалог на сервере, затем отправляем вопрос.
      const conv = await conversationApi.create({ title: deriveTitle(question) })
      const created = 'conversation' in conv ? conv.conversation : conv
      const now = Date.now()
      upsertConversation({
        id: created.id,
        title: created.title,
        createdAt: Date.parse(created.createdAt) || now,
        updatedAt: Date.parse(created.updatedAt) || now,
      })
      convId = created.id
      navigate(paths.app.chat(convId), { replace: true })
    }
    await send(convId, question)
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
