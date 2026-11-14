import DeleteOutlineIcon from '@mui/icons-material/DeleteOutline'
import EditOutlinedIcon from '@mui/icons-material/EditOutlined'
import OpenInNewIcon from '@mui/icons-material/OpenInNew'
import {
  Box,
  Chip,
  IconButton,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material'
import type { Document } from '@/entities/document'
import { formatDate } from '@/shared/lib'

interface DocumentsTableProps {
  documents: Document[]
  onView?: (doc: Document) => void
  onEdit?: (doc: Document) => void
  onDelete?: (doc: Document) => void
}

export function DocumentsTable({
  documents,
  onView,
  onEdit,
  onDelete,
}: DocumentsTableProps) {
  return (
    <TableContainer
      component={Box}
      sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 2, bgcolor: 'background.paper' }}
    >
      <Table size="small">
        <TableHead>
          <TableRow>
            <TableCell>Название</TableCell>
            <TableCell>Теги</TableCell>
            <TableCell>Ред.</TableCell>
            <TableCell>Обновлён</TableCell>
            <TableCell align="right">Действия</TableCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {documents.map((doc) => (
            <TableRow key={doc.id} hover>
              <TableCell sx={{ maxWidth: 380 }}>
                <Typography variant="body2" fontWeight={500} noWrap title={doc.title}>
                  {doc.title}
                </Typography>
              </TableCell>
              <TableCell>
                <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                  {doc.tags.slice(0, 3).map((t) => (
                    <Chip key={t} size="small" variant="outlined" label={t} />
                  ))}
                  {doc.tags.length > 3 && (
                    <Chip size="small" variant="outlined" label={`+${doc.tags.length - 3}`} />
                  )}
                </Box>
              </TableCell>
              <TableCell>{doc.version}</TableCell>
              <TableCell>{formatDate(doc.updatedAt)}</TableCell>
              <TableCell align="right" sx={{ whiteSpace: 'nowrap' }}>
                {onView && (
                  <Tooltip title="Открыть">
                    <IconButton size="small" onClick={() => onView(doc)} aria-label="Открыть документ">
                      <OpenInNewIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                {onEdit && (
                  <Tooltip title="Редактировать">
                    <IconButton size="small" onClick={() => onEdit(doc)} aria-label="Редактировать документ">
                      <EditOutlinedIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
                {onDelete && (
                  <Tooltip title="Удалить">
                    <IconButton size="small" color="error" onClick={() => onDelete(doc)} aria-label="Удалить документ">
                      <DeleteOutlineIcon fontSize="small" />
                    </IconButton>
                  </Tooltip>
                )}
              </TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </TableContainer>
  )
}
