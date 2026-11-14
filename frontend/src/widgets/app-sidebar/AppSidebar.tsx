import AddCommentOutlinedIcon from '@mui/icons-material/AddCommentOutlined'
import AdminPanelSettingsOutlinedIcon from '@mui/icons-material/AdminPanelSettingsOutlined'
import ArticleOutlinedIcon from '@mui/icons-material/ArticleOutlined'
import BookmarkBorderIcon from '@mui/icons-material/BookmarkBorder'
import {
  Box,
  Divider,
  Drawer,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
} from '@mui/material'
import { NavLink } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { config } from '@/shared/config/env'

export const DRAWER_WIDTH = 248

interface AppSidebarProps {
  mobileOpen: boolean
  onMobileClose: () => void
}

const userItems = [
  { label: 'Ассистент', icon: <AddCommentOutlinedIcon />, to: paths.app.chatNew },
  { label: 'Документы', icon: <ArticleOutlinedIcon />, to: paths.app.documents },
  { label: 'Инструкции', icon: <BookmarkBorderIcon />, to: paths.app.instructions },
]

export function AppSidebar({ mobileOpen, onMobileClose }: AppSidebarProps) {
  const content = (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Toolbar sx={{ px: 2 }}>
        <Box>
          <Typography variant="h6" fontWeight={700} color="primary.main">
            {config.appName}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            навигатор по регламентам МФТИ
          </Typography>
        </Box>
      </Toolbar>
      <Divider />
      <List sx={{ px: 1.5, py: 2, flex: 1 }}>
        {userItems.map((item) => (
          <ListItemButton
            key={item.to}
            component={NavLink}
            to={item.to}
            onClick={onMobileClose}
            sx={{ mb: 0.5 }}
          >
            <ListItemIcon sx={{ minWidth: 40 }}>{item.icon}</ListItemIcon>
            <ListItemText primary={item.label} />
          </ListItemButton>
        ))}
      </List>
      {config.enableAdmin && (
        <>
          <Divider />
          <List sx={{ px: 1.5, py: 2 }}>
            <ListItemButton
              component={NavLink}
              to={paths.admin.root}
              onClick={onMobileClose}
            >
              <ListItemIcon sx={{ minWidth: 40 }}>
                <AdminPanelSettingsOutlinedIcon />
              </ListItemIcon>
              <ListItemText primary="Администрирование" />
            </ListItemButton>
          </List>
        </>
      )}
    </Box>
  )

  return (
    <>
      {/* Desktop: постоянная панель */}
      <Drawer
        variant="permanent"
        sx={{
          display: { xs: 'none', md: 'block' },
          width: DRAWER_WIDTH,
          flexShrink: 0,
          '& .MuiDrawer-paper': { width: DRAWER_WIDTH, borderRight: '1px solid', borderColor: 'divider' },
        }}
      >
        {content}
      </Drawer>
      {/* Mobile: выдвижная панель */}
      <Drawer
        variant="temporary"
        open={mobileOpen}
        onClose={onMobileClose}
        ModalProps={{ keepMounted: true }}
        sx={{ display: { xs: 'block', md: 'none' }, '& .MuiDrawer-paper': { width: DRAWER_WIDTH } }}
      >
        {content}
      </Drawer>
    </>
  )
}
