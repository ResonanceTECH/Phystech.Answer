import ConstructionIcon from '@mui/icons-material/Construction'
import { Box, Button, Paper, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'

interface Props {
  title: string
  description: string
}

// Заглушка для разделов админки, для которых backend ещё не предоставляет API.
export function AdminStubPage({ title, description }: Props) {
  return (
    <Box sx={{ maxWidth: 640 }}>
      <Paper variant="outlined" sx={{ p: { xs: 3, md: 5 }, textAlign: 'center' }}>
        <ConstructionIcon sx={{ fontSize: 48, color: 'text.secondary', mb: 2 }} />
        <Typography variant="h6" gutterBottom>
          {title}
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 3 }}>
          {description}
        </Typography>
        <Button component={RouterLink} to={paths.admin.dashboard} variant="outlined">
          На главную админки
        </Button>
      </Paper>
    </Box>
  )
}
