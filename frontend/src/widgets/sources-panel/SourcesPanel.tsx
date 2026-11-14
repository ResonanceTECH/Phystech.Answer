import CloseIcon from '@mui/icons-material/Close'
import {
  Alert,
  Box,
  Chip,
  CircularProgress,
  Divider,
  IconButton,
  Paper,
  Tooltip,
  Typography,
} from '@mui/material'
import { useDocument } from '@/entities/document'
import { useChatUiStore } from '@/shared/lib'
import { formatDate } from '@/shared/lib'
import { EmptyState, ErrorState } from '@/shared/ui'

// Правая панель чата: просмотр документа-источника и цитаты,
// подтверждающей ответ. Открывается по клику на источник в сообщении.
export function SourcesPanel() {
  const { selectedDocumentId, selectedQuote, closeSources } = useChatUiStore()
  const document = useDocument(selectedDocumentId ?? undefined)

  return (
    <Box
      sx={{
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        bgcolor: 'background.paper',
      }}
    >
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          px: 2,
          py: 1.25,
          borderBottom: '1px solid',
          borderColor: 'divider',
        }}
      >
        <Typography variant="subtitle2" noWrap sx={{ fontWeight: 600 }}>
          Источник
        </Typography>
        <Tooltip title="Закрыть панель">
          <IconButton size="small" onClick={closeSources} aria-label="Закрыть панель источников">
            <CloseIcon fontSize="small" />
          </IconButton>
        </Tooltip>
      </Box>

      <Box sx={{ flex: 1, overflowY: 'auto', p: 2 }}>
        {!selectedDocumentId ? (
          <EmptyState
            title="Источник не выбран"
            description="Нажмите на источник под ответом ассистента, чтобы открыть соответствующий документ."
          />
        ) : document.isPending ? (
          <Box sx={{ display: 'flex', justifyContent: 'center', py: 6 }}>
            <CircularProgress size={28} />
          </Box>
        ) : document.isError ? (
          <ErrorState error={document.error} title="Не удалось открыть документ" />
        ) : document.data ? (
          <DocumentSource doc={document.data} quote={selectedQuote} />
        ) : null}
      </Box>
    </Box>
  )
}

function DocumentSource({
  doc,
  quote,
}: {
  doc: { title: string; content: string; tags: string[]; version: number; updatedAt: string }
  quote: string | null
}) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <Box>
        <Typography variant="h6" sx={{ fontSize: '1.05rem', fontWeight: 600 }}>
          {doc.title}
        </Typography>
        <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', mt: 1 }}>
          <Chip size="small" variant="outlined" label={`ред. ${doc.version}`} />
          <Chip size="small" variant="outlined" label={formatDate(doc.updatedAt)} />
          {doc.tags.map((t) => (
            <Chip key={t} size="small" variant="outlined" color="default" label={t} />
          ))}
        </Box>
      </Box>

      {quote && (
        <Alert severity="info" variant="outlined" sx={{ whiteSpace: 'pre-wrap' }}>
          <Typography variant="body2">{quote}</Typography>
        </Alert>
      )}

      <Divider />

      {/* Текст документа как лист бумаги: читается при любой теме */}
      <Paper variant="outlined" sx={{ p: 2, bgcolor: '#fff' }}>
        <Typography
          variant="body2"
          sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.65, color: '#1f1a2e' }}
        >
          {doc.content}
        </Typography>
      </Paper>
    </Box>
  )
}
