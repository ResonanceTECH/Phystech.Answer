import BookmarkBorderIcon from '@mui/icons-material/BookmarkBorder'
import BookmarkAddedIcon from '@mui/icons-material/BookmarkAdded'
import {
  Box,
  Button,
  Chip,
  Paper,
  Skeleton,
  Tooltip,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import type { ChatMessage } from '@/entities/conversation'
import { useSaveInstruction } from '@/features/save-instruction'
import { useChatUiStore } from '@/shared/lib'
import { formatTime } from '@/shared/lib'
import type { StatusKind } from '@/shared/ui'

const STATUS_LABEL: Record<string, { kind: StatusKind; label: string }> = {
  answered: { kind: 'success', label: 'Ответ подтверждён источниками' },
  partial: { kind: 'warning', label: 'Частичный ответ' },
  clarification_required: { kind: 'info', label: 'Нужен уточняющий вопрос' },
  not_found: { kind: 'warning', label: 'Информация не найдена' },
  conflict: { kind: 'warning', label: 'Противоречия в документах' },
  error: { kind: 'error', label: 'Ошибка' },
}

interface MessageBubbleProps {
  message: ChatMessage
  conversationTitle?: string
}

export function MessageBubble({ message, conversationTitle }: MessageBubbleProps) {
  if (message.role === 'user') {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
        <Paper
          elevation={0}
          sx={{
            maxWidth: '78%',
            px: 2,
            py: 1.25,
            bgcolor: 'primary.main',
            color: 'primary.contrastText',
            borderRadius: 3,
            borderBottomRightRadius: 1,
          }}
        >
          <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.55 }}>
            {message.text}
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.7, display: 'block', mt: 0.5, textAlign: 'right' }}>
            {formatTime(message.createdAt)}
          </Typography>
        </Paper>
      </Box>
    )
  }

  return <AssistantBubble message={message} conversationTitle={conversationTitle} />
}

function AssistantBubble({
  message,
  conversationTitle,
}: {
  message: ChatMessage
  conversationTitle?: string
}) {
  const openSources = useChatUiStore((s) => s.openSources)
  const saveInstruction = useSaveInstruction()
  const [saved, setSaved] = useState(false)

  if (message.pending) {
    return (
      <Box sx={{ maxWidth: '85%' }}>
        <Paper
          elevation={0}
          sx={{
            px: 2,
            py: 1.5,
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 3,
            borderBottomLeftRadius: 1,
            bgcolor: 'background.paper',
          }}
        >
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Ищу подходящие документы и формирую ответ…
          </Typography>
          <Skeleton variant="text" width="90%" />
          <Skeleton variant="text" width="75%" />
          <Skeleton variant="text" width="60%" />
        </Paper>
      </Box>
    )
  }

  const status = message.status ? STATUS_LABEL[message.status] : undefined
  const sources = message.sources ?? []

  const handleSave = () => {
    saveInstruction({
      title: conversationTitle ?? 'Ответ ассистента',
      answer: message.text,
      sources,
    })
    setSaved(true)
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, maxWidth: '92%' }}>
      <Paper
        elevation={0}
        sx={{
          px: 2,
          py: 1.5,
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 3,
          borderBottomLeftRadius: 1,
          bgcolor: 'background.paper',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1, flexWrap: 'wrap' }}>
          {status && <Chip size="small" variant="outlined" color={status.kind} label={status.label} />}
          {message.stub && (
            <Chip size="small" variant="outlined" color="warning" label="демо-ответ: ML недоступен" />
          )}
          <Typography variant="caption" color="text.secondary" sx={{ ml: 'auto' }}>
            {formatTime(message.createdAt)}
          </Typography>
        </Box>
        <Typography
          variant="body2"
          component="div"
          sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}
        >
          {message.text}
        </Typography>

        {sources.length > 0 && (
          <Box sx={{ mt: 1.5, display: 'flex', flexWrap: 'wrap', gap: 0.75, alignItems: 'center' }}>
            <Typography variant="caption" color="text.secondary" fontWeight={600}>
              ИСТОЧНИКИ:
            </Typography>
            {sources.map((src, i) => (
              <Tooltip key={src.id} title={src.quote} arrow>
                <Chip
                  size="small"
                  variant="outlined"
                  color="primary"
                  label={`${i + 1}. ${src.documentTitle}${src.sectionNumber ? ` · п. ${src.sectionNumber}` : ''}`}
                  onClick={() => openSources(src.documentId, src.quote)}
                  sx={{ maxWidth: 340 }}
                />
              </Tooltip>
            ))}
          </Box>
        )}

        {!message.stub && (
          <Box sx={{ mt: 1.5 }}>
            <Button
              size="small"
              variant="text"
              startIcon={saved ? <BookmarkAddedIcon /> : <BookmarkBorderIcon />}
              onClick={handleSave}
              disabled={saved}
            >
              {saved ? 'Сохранено' : 'Сохранить инструкцию'}
            </Button>
          </Box>
        )}
      </Paper>
    </Box>
  )
}
