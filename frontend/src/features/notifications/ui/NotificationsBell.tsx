import AssignmentIndOutlinedIcon from '@mui/icons-material/AssignmentIndOutlined'
import FiberNewOutlinedIcon from '@mui/icons-material/FiberNewOutlined'
import NotificationsNoneOutlinedIcon from '@mui/icons-material/NotificationsNoneOutlined'
import UpdateOutlinedIcon from '@mui/icons-material/UpdateOutlined'
import { Badge, Box, Button, Divider, IconButton, Popover, Stack, Tooltip, Typography } from '@mui/material'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { useMarkNotificationsRead, useNotifications, type AppNotification } from '../api/notificationsApi'

const ICON = {
  assigned: <AssignmentIndOutlinedIcon color="primary" fontSize="small" />,
  new_document: <FiberNewOutlinedIcon color="warning" fontSize="small" />,
  updated: <UpdateOutlinedIcon color="info" fontSize="small" />,
}

function ago(iso: string) {
  const min = Math.round((Date.now() - Date.parse(iso)) / 60000)
  if (min < 1) return 'только что'
  if (min < 60) return `${min} мин назад`
  const h = Math.round(min / 60)
  if (h < 24) return `${h} ч назад`
  return new Date(iso).toLocaleDateString('ru-RU')
}

// Уведомления сотрудника: назначенные документы, новые и изменённые
// документы на платформе.
export function NotificationsBell() {
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const { data } = useNotifications(true)
  const markRead = useMarkNotificationsRead()
  const navigate = useNavigate()
  const items = data?.items ?? []

  const open = (n: AppNotification) => {
    if (!n.readAt) markRead.mutate(n.id)
    setAnchor(null)
    if (n.kind === 'assigned') navigate(paths.app.plan)
    else if (n.documentId) navigate(paths.app.document(n.documentId))
  }

  return (
    <>
      <Tooltip title="Уведомления">
        <IconButton onClick={(e) => setAnchor(e.currentTarget)} aria-label={`Уведомления: ${data?.unread ?? 0} непрочитанных`}>
          <Badge color="error" badgeContent={data?.unread ?? 0} max={99}>
            <NotificationsNoneOutlinedIcon />
          </Badge>
        </IconButton>
      </Tooltip>
      <Popover
        open={Boolean(anchor)}
        anchorEl={anchor}
        onClose={() => setAnchor(null)}
        anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
        transformOrigin={{ vertical: 'top', horizontal: 'right' }}
        slotProps={{ paper: { sx: { width: 380, maxHeight: 480, borderRadius: 3 } } }}
      >
        <Stack direction="row" sx={{ alignItems: 'center', px: 2, py: 1.25 }}>
          <Typography variant="subtitle1" sx={{ fontWeight: 700, flex: 1 }}>
            Уведомления
          </Typography>
          {(data?.unread ?? 0) > 0 && (
            <Button size="small" onClick={() => markRead.mutate(undefined)}>
              Прочитать все
            </Button>
          )}
        </Stack>
        <Divider />
        {items.length === 0 ? (
          <Typography color="text.secondary" variant="body2" sx={{ p: 3, textAlign: 'center' }}>
            Новых уведомлений нет
          </Typography>
        ) : (
          items.map((n) => (
            <Box
              key={n.id}
              component="button"
              type="button"
              onClick={() => open(n)}
              sx={{
                display: 'flex',
                gap: 1.25,
                width: '100%',
                textAlign: 'left',
                border: 0,
                borderBottom: '1px solid',
                borderColor: 'divider',
                px: 2,
                py: 1.25,
                cursor: 'pointer',
                font: 'inherit',
                color: 'text.primary',
                bgcolor: n.readAt ? 'transparent' : 'rgba(139,92,246,0.07)',
                '&:hover': { bgcolor: 'action.hover' },
              }}
            >
              <Box sx={{ pt: 0.25 }}>{ICON[n.kind]}</Box>
              <Box sx={{ minWidth: 0 }}>
                <Typography variant="body2" sx={{ fontWeight: n.readAt ? 500 : 700 }}>
                  {n.title}
                </Typography>
                {n.body && (
                  <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                    {n.body}
                  </Typography>
                )}
                <Typography variant="caption" color="text.disabled">
                  {ago(n.createdAt)}
                </Typography>
              </Box>
            </Box>
          ))
        )}
      </Popover>
    </>
  )
}
