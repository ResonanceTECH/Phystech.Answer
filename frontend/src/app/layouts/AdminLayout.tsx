import { Box } from '@mui/material'
import { Outlet } from 'react-router-dom'
import { AppHeader } from '@/widgets/app-header/AppHeader'
import { AdminSidebar } from '@/widgets/admin-sidebar/AdminSidebar'
import { useAppUiStore } from '@/shared/lib'

// Макет административной части: своё боковое меню + общая шапка.
export function AdminLayout() {
  const mobileNavOpen = useAppUiStore((s) => s.mobileNavOpen)
  const setMobileNavOpen = useAppUiStore((s) => s.setMobileNavOpen)

  return (
    <Box sx={{ display: 'flex', minHeight: '100svh' }}>
      <AdminSidebar
        mobileOpen={mobileNavOpen}
        onMobileClose={() => setMobileNavOpen(false)}
      />
      <Box
        component="main"
        sx={{ flex: 1, minWidth: 0, display: 'flex', flexDirection: 'column' }}
      >
        <AppHeader title="Администрирование" />
        <Box sx={{ flex: 1, p: { xs: 2, md: 3 } }}>
          <Outlet />
        </Box>
      </Box>
    </Box>
  )
}
