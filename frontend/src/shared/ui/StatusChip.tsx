import { Chip, type ChipProps } from '@mui/material'

export type StatusKind =
  | 'default'
  | 'success'
  | 'warning'
  | 'error'
  | 'info'
  | 'read'

const KIND_COLOR: Record<StatusKind, ChipProps['color']> = {
  default: 'default',
  success: 'success',
  warning: 'warning',
  error: 'error',
  info: 'info',
  read: 'success',
}

interface StatusChipProps {
  kind?: StatusKind
  label: string
  size?: ChipProps['size']
  variant?: ChipProps['variant']
}

export function StatusChip({
  kind = 'default',
  label,
  size = 'small',
  variant = 'outlined',
}: StatusChipProps) {
  return (
    <Chip
      size={size}
      variant={variant}
      color={KIND_COLOR[kind]}
      label={label}
      sx={{ fontWeight: 500 }}
    />
  )
}
