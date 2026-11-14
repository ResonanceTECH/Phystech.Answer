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
    <Box sx={{ display: 'flex', minHeight: '100svh' }}>
      <AppSidebar
        mobileOpen={mobileNavOpen}
        onMobileClose={() => setMobileNavOpen(false)}
      />
      <Box
        component="main"
        sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}
      >
        <AppHeader />
        <Box sx={{ flex: 1, p: { xs: 2, md: 3 } }}>
          <Outlet />
        </Box>
      </Box>
    </Box>
  )
}
