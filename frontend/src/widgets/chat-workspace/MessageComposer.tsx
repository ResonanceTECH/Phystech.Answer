import SendIcon from '@mui/icons-material/Send'
import { Box, IconButton, TextField } from '@mui/material'
import { useState, type KeyboardEvent } from 'react'

interface MessageComposerProps {
  disabled?: boolean
  onSend: (text: string) => void
}

export function MessageComposer({ disabled, onSend }: MessageComposerProps) {
  const [text, setText] = useState('')

  const submit = () => {
    const value = text.trim()
    if (!value || disabled) return
    onSend(value)
    setText('')
  }

  const onKeyDown = (e: KeyboardEvent<HTMLDivElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      submit()
    }
  }

  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'flex-end',
        gap: 1,
        p: 1.5,
        borderTop: '1px solid',
        borderColor: 'divider',
        bgcolor: 'background.paper',
      }}
    >
      <TextField
        fullWidth
        multiline
        maxRows={6}
        size="small"
        placeholder="Опишите вашу задачу обычными словами…"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={onKeyDown}
        disabled={disabled}
      />
      <IconButton
        color="primary"
        onClick={submit}
        disabled={disabled || !text.trim()}
        aria-label="Отправить"
        sx={{ mb: 0.25 }}
      >
        <SendIcon />
      </IconButton>
    </Box>
  )
}
