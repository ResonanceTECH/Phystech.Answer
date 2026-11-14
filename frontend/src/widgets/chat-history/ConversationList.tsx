import AddIcon from '@mui/icons-material/Add'
import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined'
import {
  Box,
  Button,
  IconButton,
  List,
  ListItemButton,
  ListItemText,
  Tooltip,
  Typography,
} from '@mui/material'
import { useNavigate } from 'react-router-dom'
import { conversationApi, useConversationStore } from '@/entities/conversation'
import { paths } from '@/app/router/routePaths'
import { formatDate } from '@/shared/lib'

interface ConversationListProps {
  activeId?: string
  onNavigate?: () => void
}

export function ConversationList({ activeId, onNavigate }: ConversationListProps) {
  const conversations = useConversationStore((s) => s.conversations)
  const deleteConversation = useConversationStore((s) => s.deleteConversation)
  const navigate = useNavigate()

  const go = (path: string) => {
    navigate(path)
    onNavigate?.()
  }

  return (
    <Box
      sx={{
        width: 260,
        flexShrink: 0,
        borderRight: '1px solid',
        borderColor: 'divider',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'background.paper',
        height: '100%',
      }}
    >
      <Box sx={{ p: 1.5 }}>
        <Button
          fullWidth
          variant="contained"
          startIcon={<AddIcon />}
          onClick={() => go(paths.app.chatNew)}
        >
          Новый диалог
        </Button>
      </Box>
      <Box sx={{ px: 1.5, pb: 0.5 }}>
        <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
          ИСТОРИЯ ДИАЛОГОВ
        </Typography>
      </Box>
      <List dense sx={{ flex: 1, overflowY: 'auto', px: 1 }}>
        {conversations.length === 0 && (
          <Typography variant="body2" color="text.secondary" sx={{ px: 1.5, py: 2 }}>
            Диалогов пока нет — задайте первый вопрос.
          </Typography>
        )}
        {conversations.map((c) => (
          <ListItemButton
            key={c.id}
            selected={c.id === activeId}
            onClick={() => go(paths.app.chat(c.id))}
            sx={{ borderRadius: 1.5, mb: 0.25, pr: 0.5 }}
          >
            <ListItemText
              primary={c.title}
              secondary={formatDate(new Date(c.updatedAt))}
              slotProps={{
                primary: {
                  variant: 'body2',
                  noWrap: true,
                  sx: { fontWeight: c.id === activeId ? 600 : 400 },
                },
                secondary: { variant: 'caption' },
              }}
            />
            <Tooltip title="Удалить диалог">
              <IconButton
                size="small"
                edge="end"
                aria-label="Удалить диалог"
                onClick={(e) => {
                  e.stopPropagation()
                  void conversationApi.remove(c.id).catch(() => {})
                  deleteConversation(c.id)
                  if (c.id === activeId) go(paths.app.chatNew)
                }}
              >
                <DeleteOutlinedIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </ListItemButton>
        ))}
      </List>
    </Box>
  )
}
