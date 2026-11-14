import { Box, Typography } from '@mui/material'
import { useEffect, useRef } from 'react'
import type { ChatMessage } from '@/entities/conversation'
import { MessageBubble } from './MessageBubble'

interface ChatMessageListProps {
  messages: ChatMessage[]
  conversationTitle?: string
}

export function ChatMessageList({ messages, conversationTitle }: ChatMessageListProps) {
  const bottomRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' })
  }, [messages.length])

  if (messages.length === 0) {
    return (
      <Box
        sx={{
          flex: 1,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          px: 3,
        }}
      >
        <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 460, textAlign: 'center' }}>
          Задайте вопрос о регламентах и административных процедурах МФТИ —
          например, как оформить командировку или открыть финансово-лицевой счёт.
        </Typography>
      </Box>
    )
  }

  return (
    <Box
      sx={{
        flex: 1,
        overflowY: 'auto',
        px: { xs: 1.5, md: 3 },
        py: 2,
      }}
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, maxWidth: 860, mx: 'auto' }}>
        {messages.map((m) => (
          <MessageBubble key={m.id} message={m} conversationTitle={conversationTitle} />
        ))}
        <div ref={bottomRef} />
      </Box>
    </Box>
  )
}
