import { Box, Skeleton } from '@mui/material'

interface LoadingStateProps {
  rows?: number
}

// Блочные скелетоны вместо пустой области во время загрузки.
export function LoadingState({ rows = 4 }: LoadingStateProps) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5, py: 2 }}>
      {Array.from({ length: rows }, (_, i) => (
        <Skeleton
          key={i}
          variant="rounded"
          height={i === 0 ? 56 : 40}
          sx={{ borderRadius: 2 }}
        />
      ))}
    </Box>
  )
}
