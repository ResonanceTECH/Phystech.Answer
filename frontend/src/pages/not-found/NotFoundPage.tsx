import SearchOffIcon from '@mui/icons-material/SearchOff'
import { Box, Button, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'

export function NotFoundPage() {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 2,
        py: 12,
        textAlign: 'center',
      }}
    >
      <SearchOffIcon sx={{ fontSize: 56, color: 'text.secondary' }} />
      <Typography variant="h5">Страница не найдена</Typography>
      <Typography variant="body2" color="text.secondary">
        Возможно, страница была перемещена или удалена.
      </Typography>
      <Button component={RouterLink} to={paths.app.root} variant="contained">
        На главную
      </Button>
    </Box>
  )
}
