import InboxOutlinedIcon from '@mui/icons-material/InboxOutlined'
import { Box, Typography } from '@mui/material'
import type { ReactNode } from 'react'

interface EmptyStateProps {
  title: string
  description?: string
  action?: ReactNode
}

export function EmptyState({ title, description, action }: EmptyStateProps) {
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
        color: 'text.secondary',
      }}
    >
      <InboxOutlinedIcon sx={{ fontSize: 48, opacity: 0.5 }} />
      <Typography variant="h6" sx={{ color: 'text.primary' }}>
        {title}
      </Typography>
      {description && (
        <Typography variant="body2" sx={{ maxWidth: 420 }}>
          {description}
        </Typography>
      )}
      {action}
    </Box>
  )
}
