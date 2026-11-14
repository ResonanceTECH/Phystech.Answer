import AdminPanelSettingsOutlinedIcon from '@mui/icons-material/AdminPanelSettingsOutlined'
import LogoutIcon from '@mui/icons-material/Logout'
import MenuIcon from '@mui/icons-material/Menu'
import RouteOutlinedIcon from '@mui/icons-material/RouteOutlined'
import {
  AppBar,
  Avatar,
  Box,
  Divider,
  IconButton,
  ListItemIcon,
  Menu,
  MenuItem,
  Toolbar,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { ROLE_LABEL, canManageDocuments, useLogout, useSession } from '@/entities/session'
import { config } from '@/shared/config/env'
import { useAppUiStore } from '@/shared/lib'

interface AppHeaderProps {
  title?: string
}

function initials(name: string) {
  return name
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((p) => p[0]?.toUpperCase())
    .join('')
}

export function AppHeader({ title }: AppHeaderProps) {
  const setMobileNavOpen = useAppUiStore((s) => s.setMobileNavOpen)
  const { user } = useSession()
  const logout = useLogout()
  const navigate = useNavigate()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)

  const go = (to: string) => {
    setAnchor(null)
    navigate(to)
  }

  return (
    <AppBar
      position="sticky"
      elevation={0}
      sx={{
        bgcolor: 'background.paper',
        color: 'text.primary',
        borderBottom: '1px solid',
        borderColor: 'divider',
      }}
    >
      <Toolbar sx={{ gap: 1 }}>
        <IconButton
          edge="start"
          onClick={() => setMobileNavOpen(true)}
          sx={{ display: { md: 'none' } }}
          aria-label="Открыть меню"
        >
          <MenuIcon />
        </IconButton>
        <Typography variant="h6" noWrap sx={{ flexGrow: 1, fontWeight: 600 }}>
          {title ?? config.appName}
        </Typography>

        {user && (
          <>
            <Box
              component="button"
              type="button"
              onClick={(e) => setAnchor(e.currentTarget)}
              aria-label="Меню пользователя"
              sx={{
                display: 'flex',
                alignItems: 'center',
                gap: 1.25,
                border: 0,
                bgcolor: 'transparent',
                cursor: 'pointer',
                borderRadius: 999,
                pl: 0.5,
                pr: { xs: 0.5, sm: 1.5 },
                py: 0.5,
                color: 'inherit',
                font: 'inherit',
                '&:hover': { bgcolor: 'action.hover' },
              }}
            >
              <Avatar sx={{ width: 34, height: 34, fontSize: 14, bgcolor: 'primary.main' }}>
                {initials(user.name)}
              </Avatar>
              <Box sx={{ textAlign: 'left', display: { xs: 'none', sm: 'block' } }}>
                <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.2 }}>
                  {user.name}
                </Typography>
                <Typography variant="caption" color="text.secondary" sx={{ lineHeight: 1.2 }}>
                  {ROLE_LABEL[user.role] ?? user.role}
                </Typography>
              </Box>
            </Box>
            <Menu
              anchorEl={anchor}
              open={Boolean(anchor)}
              onClose={() => setAnchor(null)}
              anchorOrigin={{ vertical: 'bottom', horizontal: 'right' }}
              transformOrigin={{ vertical: 'top', horizontal: 'right' }}
            >
              <MenuItem onClick={() => go(paths.app.onboarding)}>
                <ListItemIcon>
                  <RouteOutlinedIcon fontSize="small" />
                </ListItemIcon>
                Мой план ознакомления
              </MenuItem>
              {canManageDocuments(user) && (
                <MenuItem onClick={() => go(paths.admin.root)}>
                  <ListItemIcon>
                    <AdminPanelSettingsOutlinedIcon fontSize="small" />
                  </ListItemIcon>
                  Администрирование
                </MenuItem>
              )}
              <Divider />
              <MenuItem
                onClick={() => {
                  setAnchor(null)
                  logout.mutate(undefined, { onSettled: () => navigate(paths.auth.login, { replace: true }) })
                }}
              >
                <ListItemIcon>
                  <LogoutIcon fontSize="small" />
                </ListItemIcon>
                Выйти
              </MenuItem>
            </Menu>
          </>
        )}
      </Toolbar>
    </AppBar>
  )
}
