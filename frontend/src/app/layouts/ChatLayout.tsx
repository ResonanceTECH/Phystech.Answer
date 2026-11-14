import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import ArticleOutlinedIcon from '@mui/icons-material/ArticleOutlined'
import MenuIcon from '@mui/icons-material/Menu'
import {
  Box,
  Chip,
  Drawer,
  IconButton,
  Toolbar,
  Tooltip,
  Typography,
  useMediaQuery,
} from '@mui/material'
import { useEffect, useState } from 'react'
import { useTheme } from '@mui/material/styles'
import { Link as RouterLink, Outlet, useParams } from 'react-router-dom'
import { conversationApi, useConversationStore } from '@/entities/conversation'
import { paths } from '@/app/router/routePaths'
import { config } from '@/shared/config/env'
import { useChatUiStore } from '@/shared/lib'
import { ConversationList } from '@/widgets/chat-history/ConversationList'
import { SourcesPanel } from '@/widgets/sources-panel/SourcesPanel'

const SOURCES_WIDTH = 420

// Макет AI-чата: слева история диалогов, центр — переписка, справа —
// панель источников (на мобильных — выдвижные панели).
export function ChatLayout() {
  const theme = useTheme()
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'))
  const { conversationId } = useParams<{ conversationId: string }>()
  const sourcesOpen = useChatUiStore((s) => s.sourcesOpen)
  const closeSources = useChatUiStore((s) => s.closeSources)
  const toggleSources = useChatUiStore((s) => s.toggleSources)

  const [historyOpen, setHistoryOpen] = useState(false)

  const conversations = useConversationStore((s) => s.conversations)
  const setConversations = useConversationStore((s) => s.setConversations)
  const activeTitle = conversations.find((c) => c.id === conversationId)?.title

  // Сервер — источник истины для списка диалогов.
  useEffect(() => {
    let cancelled = false
    conversationApi
      .list()
      .then((list) => {
        if (cancelled) return
        setConversations(
          list.map((c) => ({
            id: c.id,
            title: c.title,
            createdAt: Date.parse(c.createdAt),
            updatedAt: Date.parse(c.updatedAt),
          })),
        )
      })
      .catch(() => {
        // Сеть/сессия — остаётся локальный кэш.
      })
    return () => {
      cancelled = true
    }
  }, [setConversations])

  return (
    <Box sx={{ display: 'flex', height: '100%', minHeight: 0, overflow: 'hidden' }}>
      {/* Левая панель: история диалогов (desktop) */}
      {isDesktop && <ConversationList activeId={conversationId} />}

      {/* Центральная область */}
      <Box
        component="main"
        sx={{
          flex: 1,
          minWidth: 0,
          display: 'flex',
          flexDirection: 'column',
          borderRight:
            isDesktop && sourcesOpen ? '1px solid' : undefined,
          borderColor: 'divider',
        }}
      >
        <Toolbar
          variant="dense"
          sx={{
            gap: 0.5,
            borderBottom: '1px solid',
            borderColor: 'divider',
            bgcolor: 'background.paper',
          }}
        >
          <IconButton
            onClick={() => setHistoryOpen(true)}
            sx={{ display: { md: 'none' } }}
            aria-label="История диалогов"
          >
            <MenuIcon />
          </IconButton>
          <Typography variant="subtitle1" noWrap sx={{ flex: 1, fontWeight: 600 }}>
            {activeTitle ?? 'Ассистент'}
          </Typography>
          {config.demoMode && (
            <Chip size="small" color="warning" variant="outlined" label="Демо" />
          )}
          <Tooltip title={sourcesOpen ? 'Скрыть источники' : 'Показать источники'}>
            <IconButton
              onClick={toggleSources}
              aria-label="Панель источников"
              color={sourcesOpen ? 'primary' : 'default'}
            >
              <ArticleOutlinedIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title="К списку диалогов">
            <IconButton
              component={RouterLink}
              to={paths.app.chatNew}
              aria-label="Новый диалог"
            >
              <ArrowBackIcon sx={{ transform: 'scaleX(-1)' }} />
            </IconButton>
          </Tooltip>
        </Toolbar>
        <Box sx={{ flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
          <Outlet />
        </Box>
      </Box>

      {/* Правая панель: источники (desktop) */}
      {isDesktop && sourcesOpen && (
        <Box
          sx={{
            width: SOURCES_WIDTH,
            flexShrink: 0,
            borderLeft: '1px solid',
            borderColor: 'divider',
          }}
        >
          <SourcesPanel />
        </Box>
      )}

      {/* Mobile: история диалогов */}
      <Drawer
        open={historyOpen}
        onClose={() => setHistoryOpen(false)}
        ModalProps={{ keepMounted: true }}
        sx={{ display: { md: 'none' }, '& .MuiDrawer-paper': { width: 280 } }}
      >
        <ConversationList
          activeId={conversationId}
          onNavigate={() => setHistoryOpen(false)}
        />
      </Drawer>

      {/* Mobile: панель источников */}
      <Drawer
        anchor="right"
        open={!isDesktop && sourcesOpen}
        onClose={closeSources}
        ModalProps={{ keepMounted: true }}
        sx={{ '& .MuiDrawer-paper': { width: '100%', maxWidth: 480 } }}
      >
        <SourcesPanel />
      </Drawer>
    </Box>
  )
}
