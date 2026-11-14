import ArrowBackIcon from '@mui/icons-material/ArrowBack'
import DownloadOutlinedIcon from '@mui/icons-material/DownloadOutlined'
import ForumOutlinedIcon from '@mui/icons-material/ForumOutlined'
import SearchIcon from '@mui/icons-material/Search'
import TaskAltIcon from '@mui/icons-material/TaskAlt'
import { Box, Button, Chip, InputAdornment, Stack, TextField, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import { Link as RouterLink, useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { useDocument } from '@/entities/document'
import { formatDocDate } from '@/entities/source-reference'
import { useAcknowledge, usePlan } from '@/features/reading-plan'
import { config } from '@/shared/config/env'
import { ErrorState, LoadingState } from '@/shared/ui'
import { DocumentViewer } from '@/widgets/document-viewer/DocumentViewer'

// Карточка документа: реквизиты, просмотр как в оригинале с подсветкой
// найденного (?q= — из поиска или из цитаты ассистента), отметка
// ознакомления, вопрос ассистенту по документу.
export function DocumentDetailsPage() {
  const { documentId } = useParams<{ documentId: string }>()
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const document = useDocument(documentId)
  const plan = usePlan()
  const ack = useAcknowledge()
  const doc = document.data

  const [query, setQuery] = useState(params.get('q') ?? '')
  const [applied, setApplied] = useState(params.get('q') ?? '')
  useEffect(() => {
    const q = params.get('q') ?? ''
    setQuery(q)
    setApplied(q)
  }, [params])
  // Поиск по документу — с небольшой задержкой, чтобы не перерисовывать на каждую букву.
  useEffect(() => {
    const t = setTimeout(() => setApplied(query), 350)
    return () => clearTimeout(t)
  }, [query])

  const planItem = plan.data?.items.find((i) => i.documentId === documentId)
  const acknowledged = planItem?.read && !(planItem.readAt && Date.parse(planItem.updatedAt) > Date.parse(planItem.readAt) + 1000)

  return (
    <Box sx={{ maxWidth: 1100, mx: 'auto' }}>
      <Button component={RouterLink} to={paths.app.documents} startIcon={<ArrowBackIcon />} size="small" sx={{ mb: 2 }}>
        Все документы
      </Button>

      {document.isPending && <LoadingState />}
      {document.isError && <ErrorState error={document.error} title="Документ не найден" />}

      {doc && (
        <Stack spacing={2}>
          <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ alignItems: { md: 'flex-start' }, justifyContent: 'space-between' }}>
            <Box sx={{ minWidth: 0 }}>
              <Typography variant="h5" sx={{ fontWeight: 700, lineHeight: 1.3 }}>
                {doc.title}
              </Typography>
              <Stack direction="row" spacing={1} sx={{ mt: 1, flexWrap: 'wrap', gap: 1 }}>
                {doc.documentType && <Chip size="small" label={doc.documentType} />}
                {doc.documentNumber && <Chip size="small" variant="outlined" label={`№ ${doc.documentNumber}`} />}
                {doc.documentDate && <Chip size="small" variant="outlined" label={`от ${formatDocDate(doc.documentDate.slice(0, 10))}`} />}
                {doc.category && <Chip size="small" variant="outlined" color="primary" label={doc.category} />}
                {doc.visibility && doc.visibility !== 'active' && <Chip size="small" color="warning" label={doc.visibility === 'hidden' ? 'Скрыт' : 'В архиве'} />}
              </Stack>
            </Box>
            <Stack direction="row" spacing={1} sx={{ flexShrink: 0, flexWrap: 'wrap', gap: 1 }}>
              <Button
                variant={acknowledged ? 'outlined' : 'contained'}
                color={acknowledged ? 'success' : 'primary'}
                startIcon={<TaskAltIcon />}
                disabled={acknowledged || ack.isPending}
                onClick={() => documentId && ack.mutate(documentId)}
              >
                {acknowledged ? 'Ознакомлен' : 'Отметить: ознакомлен'}
              </Button>
              <Button
                startIcon={<ForumOutlinedIcon />}
                onClick={() =>
                  navigate(paths.app.chatNew, {
                    state: { question: `Кратко: что важно знать сотруднику из документа «${doc.title}»? Какие действия, сроки и ответственных он устанавливает?` },
                  })
                }
              >
                Спросить ассистента
              </Button>
              <Button startIcon={<DownloadOutlinedIcon />} href={`${config.apiBaseUrl}/v1/documents/${doc.id}/file`} target="_blank" rel="noopener">
                Оригинал
              </Button>
            </Stack>
          </Stack>

          <TextField
            size="small"
            placeholder="Найти в документе: слово, срок, номер пункта…"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter') {
                setApplied(query)
                setParams(query ? { q: query } : {}, { replace: true })
              }
            }}
            slotProps={{
              input: {
                startAdornment: (
                  <InputAdornment position="start">
                    <SearchIcon fontSize="small" />
                  </InputAdornment>
                ),
              },
            }}
          />

          <DocumentViewer documentId={doc.id} content={doc.content} query={applied} />
        </Stack>
      )}
    </Box>
  )
}
