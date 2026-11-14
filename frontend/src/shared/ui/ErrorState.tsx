import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline'
import { Box, Button, Typography } from '@mui/material'
import { apiErrorMessage } from '@/shared/api/errors'

interface ErrorStateProps {
  error?: unknown
  title?: string
  onRetry?: () => void
}

export function ErrorState({
  error,
  title = 'Что-то пошло не так',
  onRetry,
}: ErrorStateProps) {
  return (
    <Box
      sx={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        justifyContent: 'center',
        gap: 1.5,
        py: 8,
        px: 2,
        textAlign: 'center',
      }}
    >
      <ErrorOutlineIcon color="error" sx={{ fontSize: 48 }} />
      <Typography variant="h6" component="div">
        {title}
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ maxWidth: 480 }}>
        {apiErrorMessage(error)}
      </Typography>
      {onRetry && (
        <Button variant="outlined" onClick={onRetry} sx={{ mt: 1 }}>
          Повторить
        </Button>
      )}
    </Box>
  )
}
