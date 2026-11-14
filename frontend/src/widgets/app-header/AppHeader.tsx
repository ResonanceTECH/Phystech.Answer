import LogoutIcon from '@mui/icons-material/Logout'
import MenuIcon from '@mui/icons-material/Menu'
import {
  AppBar,
  Avatar,
  Chip,
  IconButton,
  Toolbar,
  Tooltip,
  Typography,
} from '@mui/material'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { isAdminRole, useAuthStore } from '@/entities/user'
import { NotificationsBell } from '@/features/notifications'
import { useAppUiStore } from '@/shared/lib'
import { config } from '@/shared/config/env'

interface AppHeaderProps {
  title?: string
}

function initials(firstName: string, lastName: string): string {
  return `${firstName.slice(0, 1)}${lastName.slice(0, 1)}`.toUpperCase() || '•'
}

export function AppHeader({ title }: AppHeaderProps) {
  const setMobileNavOpen = useAppUiStore((s) => s.setMobileNavOpen)
  const user = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)
  const navigate = useNavigate()

  const onLogout = async () => {
    await logout()
    navigate(paths.auth.login, { replace: true })
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
        {config.demoMode && (
          <Chip
            size="small"
            color="warning"
            variant="outlined"
            label="Демо-данные"
            title="В базе подняты демонстрационные документы и тестовые аккаунты"
          />
        )}
        {user ? (
          <>
            <NotificationsBell />
            {isAdminRole(user.role) ? (
              <Chip
                size="small"
                variant="outlined"
                label={user.role === 'expert' ? 'Эксперт' : 'Админ'}
              />
            ) : null}
            <Tooltip title="Личный кабинет">
              <IconButton
                component={RouterLink}
                to={paths.app.profile}
                size="small"
                aria-label={`Профиль: ${user.name}`}
              >
                <Avatar
                  sx={{
                    width: 34,
                    height: 34,
                    fontSize: 14,
                    bgcolor: 'primary.main',
                  }}
                >
                  {initials(user.firstName ?? '', user.lastName ?? '')}
                </Avatar>
              </IconButton>
            </Tooltip>
            <Typography
              variant="body2"
              noWrap
              sx={{ display: { xs: 'none', sm: 'block' }, maxWidth: 200 }}
            >
              {user.name}
            </Typography>
            <Tooltip title="Выйти">
              <IconButton onClick={onLogout} aria-label="Выйти из аккаунта">
                <LogoutIcon fontSize="small" />
              </IconButton>
            </Tooltip>
          </>
        ) : null}
      </Toolbar>
    </AppBar>
  )
}
