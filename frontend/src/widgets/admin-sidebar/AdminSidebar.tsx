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
import DashboardOutlinedIcon from '@mui/icons-material/DashboardOutlined'
import ArticleOutlinedIcon from '@mui/icons-material/ArticleOutlined'
import UploadFileOutlinedIcon from '@mui/icons-material/UploadFileOutlined'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import { NavLink } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { config } from '@/shared/config/env'
import { DRAWER_WIDTH } from '@/widgets/app-sidebar/AppSidebar'

interface AdminSidebarProps {
  mobileOpen: boolean
  onMobileClose: () => void
}

// Разделы «Редакции», «Противоречия», «Аналитика» скрыты, пока у них нет API:
// изменения редакций и противоречия показываются прямо в ответах ассистента.
const adminItems = [
  { label: 'Обзор', icon: <DashboardOutlinedIcon />, to: paths.admin.dashboard, end: true },
  { label: 'Документы', icon: <ArticleOutlinedIcon />, to: paths.admin.documents },
  { label: 'Загрузка', icon: <UploadFileOutlinedIcon />, to: paths.admin.upload },
  { label: 'Участники', icon: <GroupsOutlinedIcon />, to: paths.admin.participants },
]

export function AdminSidebar({ mobileOpen, onMobileClose }: AdminSidebarProps) {
  const content = (
    <Box sx={{ display: 'flex', flexDirection: 'column', height: '100%' }}>
      <Toolbar sx={{ px: 2, flexDirection: 'column', alignItems: 'flex-start', justifyContent: 'center', gap: 0.25 }}>
        <Typography variant="h6" noWrap sx={{ fontWeight: 700, color: 'primary.main', fontSize: '0.88rem', maxWidth: '100%' }}>
          Администрирование
        </Typography>
        <Typography variant="caption" color="text.secondary">
          {config.appName} · база знаний
        </Typography>
      </Toolbar>
      <Divider />
      <List sx={{ px: 1.5, py: 2, flex: 1 }}>
        {adminItems.map((item) => (
          <ListItemButton
            key={item.to}
            component={NavLink}
            to={item.to}
            end={item.end}
            onClick={onMobileClose}
            sx={{ mb: 0.5 }}
          >
            <ListItemIcon sx={{ minWidth: 40 }}>{item.icon}</ListItemIcon>
            <ListItemText primary={item.label} />
          </ListItemButton>
        ))}
      </List>
      <Divider />
      <List sx={{ px: 1.5, py: 1.5 }}>
        <ListItemButton component={NavLink} to={paths.app.root} onClick={onMobileClose}>
          <ListItemIcon sx={{ minWidth: 40 }}>
            <ArrowBackIcon />
          </ListItemIcon>
          <ListItemText primary="К приложению" />
        </ListItemButton>
      </List>
    </Box>
  )

  return (
    <>
      <Drawer
        variant="permanent"
        sx={{
          display: { xs: 'none', md: 'block' },
          width: DRAWER_WIDTH,
          flexShrink: 0,
          '& .MuiDrawer-paper': {
            width: DRAWER_WIDTH,
            borderRight: '1px solid',
            borderColor: 'divider',
          },
        }}
      >
        {content}
      </Drawer>
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
