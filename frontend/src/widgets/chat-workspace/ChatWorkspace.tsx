import { Box } from '@mui/material'
import type { ChatMessage } from '@/entities/conversation'
import { MessageComposer } from './MessageComposer'
import { ChatMessageList } from './ChatMessageList'

interface ChatWorkspaceProps {
  messages: ChatMessage[]
  conversationTitle?: string
  disabled?: boolean
  onSend: (text: string) => void
}

// Центральная область диалога: переписка + поле ввода.
export function ChatWorkspace({
  messages,
  conversationTitle,
  disabled,
  onSend,
}: ChatWorkspaceProps) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%', minHeight: 0 }}>
      <ChatMessageList messages={messages} conversationTitle={conversationTitle} />
      <MessageComposer disabled={disabled} onSend={onSend} />
    </Box>
  )
}
