import { Box } from '@mui/material'
import { Outlet } from 'react-router-dom'
import { AppHeader } from '@/widgets/app-header/AppHeader'
import { AppSidebar } from '@/widgets/app-sidebar/AppSidebar'
import { useAppUiStore } from '@/shared/lib'

// Основной макет пользовательской части: боковая навигация + шапка + контент.
export function AppLayout() {
  const mobileNavOpen = useAppUiStore((s) => s.mobileNavOpen)
  const setMobileNavOpen = useAppUiStore((s) => s.setMobileNavOpen)

  return (
    // Высота окна фиксирована: прокручивается только контент, шапка и
    // навигация на месте (в чате — только лента сообщений, поле ввода внизу).
    <Box sx={{ display: 'flex', height: '100svh', overflow: 'hidden' }}>
      <AppSidebar
        mobileOpen={mobileNavOpen}
        onMobileClose={() => setMobileNavOpen(false)}
      />
      <Box
        component="main"
        sx={{ flex: 1, minWidth: 0, minHeight: 0, display: 'flex', flexDirection: 'column' }}
      >
        <AppHeader />
        <Box sx={{ flex: 1, minHeight: 0, overflowY: 'auto', p: { xs: 2, md: 3 } }}>
          <Outlet />
        </Box>
      </Box>
    </Box>
  )
}
