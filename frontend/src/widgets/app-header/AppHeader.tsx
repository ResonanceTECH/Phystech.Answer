import MenuIcon from '@mui/icons-material/Menu'
import { AppBar, Chip, IconButton, Toolbar, Typography } from '@mui/material'
import { useAppUiStore } from '@/shared/lib'
import { config } from '@/shared/config/env'

interface AppHeaderProps {
  title?: string
}

export function AppHeader({ title }: AppHeaderProps) {
  const setMobileNavOpen = useAppUiStore((s) => s.setMobileNavOpen)

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
            label="Демо-режим"
            title="Авторизация не подключена: данные видны всем пользователям"
          />
        )}
      </Toolbar>
    </AppBar>
  )
}
