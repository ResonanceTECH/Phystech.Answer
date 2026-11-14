import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import {
  Box,
  Button,
  Chip,
  Divider,
  Paper,
  Typography,
} from '@mui/material'
import { Link as RouterLink, useNavigate, useParams } from 'react-router-dom'
import { useDocument } from '@/entities/document'
import { paths } from '@/app/router/routePaths'
import { formatDateTime } from '@/shared/lib'
import { ErrorState, LoadingState } from '@/shared/ui'

export function DocumentDetailsPage() {
  const { documentId } = useParams<{ documentId: string }>()
  const navigate = useNavigate()
  const document = useDocument(documentId)
  const doc = document.data

  return (
    <Box sx={{ maxWidth: 860 }}>
      <Button
        component={RouterLink}
        to={paths.app.documents}
        startIcon={<ArrowBackIcon />}
        size="small"
        sx={{ mb: 2 }}
      >
        Все документы
      </Button>

      {document.isPending && <LoadingState />}
      {document.isError && (
        <ErrorState error={document.error} title="Документ не найден" />
      )}

      {doc && (
        <>
          <Box
            sx={{
              display: 'flex',
              flexWrap: 'wrap',
              alignItems: 'flex-start',
              justifyContent: 'space-between',
              gap: 2,
              mb: 3,
            }}
          >
            <Box>
              <Typography variant="h5" sx={{ fontWeight: 650 }}>
                {doc.title}
              </Typography>
              <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', mt: 1.5, alignItems: 'center' }}>
                <Chip size="small" variant="outlined" label={`ред. ${doc.version}`} />
                <Chip size="small" variant="outlined" label={`обновлён ${formatDateTime(doc.updatedAt)}`} />
                {doc.tags.map((t) => (
                  <Chip key={t} size="small" variant="outlined" color="primary" label={t} />
                ))}
              </Box>
            </Box>
            <Button
              variant="contained"
              onClick={() =>
                navigate(paths.app.chatNew, {
                  state: {
                    question: `Расскажите про «${doc.title}»: порядок действий, необходимые документы, сроки и ответственные. Что нужно знать сотруднику?`,
                  },
                })
              }
            >
              Обсудить с ассистентом
            </Button>
          </Box>

          <Paper variant="outlined" sx={{ p: { xs: 2, md: 3 } }}>
            <Typography
              variant="body2"
              sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.7 }}
            >
              {doc.content}
            </Typography>
          </Paper>

          <Divider sx={{ my: 3 }} />
          <Typography variant="caption" color="text.secondary">
            ID документа: {doc.id}
          </Typography>
        </>
      )}
    </Box>
  )
}
