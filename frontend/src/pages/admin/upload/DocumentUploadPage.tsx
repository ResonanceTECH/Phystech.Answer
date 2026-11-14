import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import CloudUploadOutlinedIcon from '@mui/icons-material/CloudUploadOutlined'
import ContentCopyOutlinedIcon from '@mui/icons-material/ContentCopyOutlined'
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutlineOutlined'
import ForumOutlinedIcon from '@mui/icons-material/ForumOutlined'
import InsertDriveFileOutlinedIcon from '@mui/icons-material/InsertDriveFileOutlined'
import ReportProblemOutlinedIcon from '@mui/icons-material/ReportProblemOutlined'
import {
  Alert,
  Box,
  Button,
  Chip,
  FormControl,
  InputLabel,
  LinearProgress,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useQueryClient } from '@tanstack/react-query'
import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { useDocumentsBrief } from '@/entities/document'
import { formatDocDate } from '@/entities/source-reference'
import { FOLDERS } from '@/features/admin-kb'
import { api, apiErrorMessage, queryKeys } from '@/shared/api'
import { PageHeader } from '@/shared/ui'

// Загрузка документов администратором: файлы уходят на бэкенд, где
// проходят тот же конвейер, что и база при старте — извлечение текста
// (PDF/DOCX/XLSX, OCR сканов), реквизиты, разбиение на пункты, индексация.
// Статус каждой загрузки опрашивается до готовности.

const ACCEPT = '.pdf,.docx,.xlsx,.txt,.md'
const SUPPORTED = /\.(pdf|docx|xlsx|txt|md)$/i
const POLL_MS = 1500

type Stage = 'uploading' | 'processing' | 'ready' | 'review' | 'duplicate' | 'rejected'

interface Row {
  key: string
  name: string
  size: number
  stage: Stage
  documentId?: string
  versionId?: string
  message?: string
  meta?: { type?: string; number?: string; date?: string; sections?: number; chars?: number }
}

interface UploadResult {
  filename: string
  status: 'queued' | 'duplicate' | 'rejected'
  documentId?: string
  versionId?: string
  error?: string
}

interface VersionInfo {
  id: string
  processingStatus: string
  processingError?: string
  content?: string
}

interface DocInfo {
  documentType?: string
  documentNumber?: string
  documentDate?: string
}

const STAGE_VIEW: Record<Stage, { label: string; color: 'default' | 'info' | 'success' | 'warning' | 'error' }> = {
  uploading: { label: 'Загрузка', color: 'info' },
  processing: { label: 'Извлечение текста · OCR · индексация', color: 'info' },
  ready: { label: 'Готов — доступен ассистенту', color: 'success' },
  review: { label: 'Требует проверки', color: 'warning' },
  duplicate: { label: 'Уже в базе', color: 'default' },
  rejected: { label: 'Отклонён', color: 'error' },
}

function fmtSize(b: number) {
  return b > 1024 * 1024 ? `${(b / 1024 / 1024).toFixed(1)} МБ` : `${Math.max(1, Math.round(b / 1024))} КБ`
}

export function DocumentUploadPage() {
  const inputRef = useRef<HTMLInputElement>(null)
  const [rows, setRows] = useState<Row[]>([])
  const [dragOver, setDragOver] = useState(false)
  const [error, setError] = useState('')
  const [category, setCategory] = useState('')
  const [visibility, setVisibility] = useState<'active' | 'hidden'>('active')
  const [revisionDocumentId, setRevisionDocumentId] = useState('')
  const [assignMode, setAssignMode] = useState<'none' | 'required' | 'optional'>('none')
  const [dueDate, setDueDate] = useState('')
  const qc = useQueryClient()
  const navigate = useNavigate()
  const documents = useDocumentsBrief()

  const patch = (key: string, p: Partial<Row>) =>
    setRows((prev) => prev.map((r) => (r.key === key ? { ...r, ...p } : r)))

  const upload = async (files: File[]) => {
    setError('')
    if (revisionDocumentId && files.length !== 1) {
      setError('Для новой редакции выберите ровно один файл.')
      return
    }
    const batch: Row[] = files.map((f, i) => ({
      key: `${Date.now()}-${i}-${f.name}`,
      name: f.name,
      size: f.size,
      stage: SUPPORTED.test(f.name) ? 'uploading' : 'rejected',
      message: SUPPORTED.test(f.name) ? undefined : 'Формат не поддерживается: PDF, DOCX, XLSX, TXT',
    }))
    setRows((prev) => [...batch, ...prev])

    const valid = files.filter((f) => SUPPORTED.test(f.name))
    if (valid.length === 0) return
    const form = new FormData()
    valid.forEach((f) => form.append('files', f, f.name))
    if (category) form.append('category', category)
    form.append('visibility', visibility)
    if (revisionDocumentId) form.append('documentId', revisionDocumentId)
    try {
      const resp = await api.postForm<{ results: UploadResult[] }>('/v1/admin/documents/upload', form)
      for (const res of resp.results) {
        const row = batch.find((r) => r.name === res.filename && r.stage === 'uploading')
        if (!row) continue
        if (res.status === 'queued') {
          patch(row.key, { stage: 'processing', documentId: res.documentId, versionId: res.versionId })
          if (assignMode !== 'none' && res.documentId) {
            await api.post('/v1/admin/assignments', {
              documentIds: [res.documentId],
              targetType: 'all',
              requirement: assignMode,
              dueDate: assignMode === 'required' && dueDate ? dueDate : undefined,
            })
          }
        } else if (res.status === 'duplicate') {
          patch(row.key, { stage: 'duplicate', message: 'Такой файл уже загружен — повторная обработка не нужна' })
        } else {
          patch(row.key, { stage: 'rejected', message: res.error ?? 'Файл отклонён' })
        }
      }
    } catch (e) {
      setError(apiErrorMessage(e))
      batch.forEach((r) => r.stage === 'uploading' && patch(r.key, { stage: 'rejected', message: 'Не удалось загрузить' }))
    }
  }

  // Опрос статуса разбора для файлов в обработке.
  const processing = rows.filter((r) => r.stage === 'processing' && r.documentId)
  useEffect(() => {
    if (processing.length === 0) return
    const timer = setInterval(async () => {
      for (const r of processing) {
        try {
          const versions = await api.get<VersionInfo[]>(`/v1/documents/${r.documentId}/versions`)
          const v = versions.find((x) => x.id === r.versionId) ?? versions[0]
          if (!v) continue
          if (v.processingStatus === 'published') {
            const [doc, sections] = await Promise.all([
              api.get<DocInfo>(`/v1/documents/${r.documentId}`),
              api.get<unknown[]>(`/v1/documents/${r.documentId}/sections`).catch(() => []),
            ])
            patch(r.key, {
              stage: 'ready',
              meta: {
                type: doc.documentType,
                number: doc.documentNumber,
                date: doc.documentDate,
                sections: sections.length,
              },
            })
            void qc.invalidateQueries({ queryKey: queryKeys.documents.all })
          } else if (v.processingStatus === 'requires_review' || v.processingStatus === 'failed') {
            patch(r.key, { stage: 'review', message: v.processingError || 'Не удалось извлечь текст' })
          }
        } catch {
          /* следующая попытка на следующем тике */
        }
      }
    }, POLL_MS)
    return () => clearInterval(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [processing.map((r) => r.key).join('|')])

  const onFiles = (list: FileList | null) => {
    if (list && list.length > 0) void upload(Array.from(list))
  }

  const reprocess = async (row: Row) => {
    if (!row.documentId) return
    setError('')
    try {
      await api.post(`/v1/admin/documents/${row.documentId}/reprocess`, {})
      patch(row.key, { stage: 'processing', message: undefined })
    } catch (e) {
      setError(apiErrorMessage(e))
    }
  }

  return (
    <Stack spacing={3} sx={{ maxWidth: 960 }}>
      <PageHeader
        title="Загрузка документов"
        subtitle="Перетащите приказы и регламенты — система сама распознает текст (включая сканы), определит реквизиты и пункты и сразу подключит документ к ассистенту."
      />

      <Paper elevation={0} sx={{ p: 2, borderRadius: 3, border: '1px solid', borderColor: 'divider' }}>
        <Stack direction={{ xs: 'column', md: 'row' }} spacing={2}>
          <FormControl size="small" sx={{ minWidth: 240, flex: 1 }}>
            <InputLabel id="revision-label">Тип загрузки</InputLabel>
            <Select
              labelId="revision-label"
              label="Тип загрузки"
              value={revisionDocumentId}
              onChange={(e) => setRevisionDocumentId(e.target.value)}
            >
              <MenuItem value="">Новый документ</MenuItem>
              {(documents.data ?? []).map((d) => (
                <MenuItem key={d.id} value={d.id}>Новая редакция: {d.title}</MenuItem>
              ))}
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 190 }} disabled={Boolean(revisionDocumentId)}>
            <InputLabel id="folder-label">Папка</InputLabel>
            <Select labelId="folder-label" label="Папка" value={category} onChange={(e) => setCategory(e.target.value)}>
              <MenuItem value="">Определить автоматически</MenuItem>
              {FOLDERS.map((f) => <MenuItem key={f} value={f}>{f}</MenuItem>)}
            </Select>
          </FormControl>
          <FormControl size="small" sx={{ minWidth: 170 }} disabled={Boolean(revisionDocumentId)}>
            <InputLabel id="visibility-label">Доступность</InputLabel>
            <Select labelId="visibility-label" label="Доступность" value={visibility} onChange={(e) => setVisibility(e.target.value as 'active' | 'hidden')}>
              <MenuItem value="active">Сразу активен</MenuItem>
              <MenuItem value="hidden">Скрыт до проверки</MenuItem>
            </Select>
          </FormControl>
        </Stack>
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={2} sx={{ mt: 2 }}>
          <FormControl size="small" sx={{ minWidth: 250 }} disabled={Boolean(revisionDocumentId)}>
            <InputLabel id="assignment-label">Назначение</InputLabel>
            <Select labelId="assignment-label" label="Назначение" value={assignMode} onChange={(e) => setAssignMode(e.target.value as typeof assignMode)}>
              <MenuItem value="none">Не назначать</MenuItem>
              <MenuItem value="required">Всем обязательно</MenuItem>
              <MenuItem value="optional">Всем для KPI</MenuItem>
            </Select>
          </FormControl>
          {assignMode === 'required' && !revisionDocumentId && (
            <TextField size="small" type="date" label="Срок ознакомления" value={dueDate} onChange={(e) => setDueDate(e.target.value)} slotProps={{ inputLabel: { shrink: true } }} />
          )}
        </Stack>
      </Paper>

      <Paper
        elevation={0}
        onDragOver={(e) => {
          e.preventDefault()
          setDragOver(true)
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={(e) => {
          e.preventDefault()
          setDragOver(false)
          onFiles(e.dataTransfer.files)
        }}
        onClick={() => inputRef.current?.click()}
        sx={{
          cursor: 'pointer',
          p: { xs: 4, md: 6 },
          textAlign: 'center',
          borderRadius: 4,
          border: '2px dashed',
          borderColor: dragOver ? 'primary.main' : 'divider',
          bgcolor: dragOver ? 'rgba(139,92,246,0.08)' : 'background.paper',
          transition: 'all 0.2s ease',
          '&:hover': { borderColor: 'primary.light', bgcolor: 'rgba(139,92,246,0.04)' },
        }}
      >
        <input
          ref={inputRef}
          type="file"
          accept={ACCEPT}
          multiple={!revisionDocumentId}
          hidden
          onChange={(e) => {
            onFiles(e.target.files)
            e.target.value = ''
          }}
        />
        <CloudUploadOutlinedIcon sx={{ fontSize: 56, color: 'primary.main', mb: 1 }} />
        <Typography variant="h6" sx={{ fontWeight: 700 }}>
          Перетащите файлы сюда или нажмите, чтобы выбрать
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
          PDF (в том числе сканы), DOCX, XLSX, TXT · можно несколько файлов сразу · до 64 МБ
        </Typography>
        <Stack direction="row" spacing={1} sx={{ justifyContent: 'center', mt: 2, flexWrap: 'wrap', gap: 1 }}>
          {['OCR сканов', 'Реквизиты: тип, №, дата', 'Разбиение на пункты', 'Повторы отсекаются'].map((t) => (
            <Chip key={t} size="small" label={t} variant="outlined" />
          ))}
        </Stack>
      </Paper>

      {error && <Alert severity="error">{error}</Alert>}

      {rows.length > 0 && (
        <Stack spacing={1.25}>
          {rows.map((r) => (
            <UploadRow
              key={r.key}
              row={r}
              onAsk={() =>
                navigate(paths.app.chatNew, {
                  state: { question: `Кратко: что устанавливает документ «${r.name.replace(/\.[^.]+$/, '')}»? Какие действия и сроки?` },
                })
              }
              onReprocess={() => void reprocess(r)}
            />
          ))}
        </Stack>
      )}
    </Stack>
  )
}

function UploadRow({ row, onAsk, onReprocess }: { row: Row; onAsk: () => void; onReprocess: () => void }) {
  const view = STAGE_VIEW[row.stage]
  const icon =
    row.stage === 'ready' ? (
      <CheckCircleIcon color="success" />
    ) : row.stage === 'review' ? (
      <ReportProblemOutlinedIcon color="warning" />
    ) : row.stage === 'rejected' ? (
      <ErrorOutlineIcon color="error" />
    ) : row.stage === 'duplicate' ? (
      <ContentCopyOutlinedIcon color="disabled" />
    ) : (
      <InsertDriveFileOutlinedIcon color="primary" />
    )
  const busy = row.stage === 'uploading' || row.stage === 'processing'
  const meta = row.meta
  const facts = meta
    ? [
        meta.type,
        meta.number ? `№ ${meta.number}` : '',
        meta.date ? `от ${formatDocDate(meta.date)}` : '',
        meta.sections ? `${meta.sections} пунктов` : '',
      ].filter(Boolean)
    : []

  return (
    <Paper elevation={0} sx={{ p: 2, borderRadius: 3, border: '1px solid', borderColor: 'divider' }}>
      <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center' }}>
        {icon}
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Typography variant="body2" sx={{ fontWeight: 600 }} noWrap title={row.name}>
            {row.name}
          </Typography>
          <Typography variant="caption" color="text.secondary">
            {fmtSize(row.size)}
            {row.message ? ` · ${row.message}` : ''}
          </Typography>
          {facts.length > 0 && (
            <Stack direction="row" spacing={0.75} sx={{ mt: 0.75, flexWrap: 'wrap', gap: 0.75 }}>
              {facts.map((f) => (
                <Chip key={f} size="small" label={f} color="primary" variant="outlined" sx={{ height: 22 }} />
              ))}
            </Stack>
          )}
        </Box>
        <Chip size="small" color={view.color} label={view.label} sx={{ fontWeight: 600 }} />
        {row.stage === 'ready' && (
          <Button size="small" startIcon={<ForumOutlinedIcon />} onClick={onAsk}>
            Спросить
          </Button>
        )}
        {row.stage === 'review' && row.documentId && (
          <Button size="small" onClick={onReprocess}>
            Обработать снова
          </Button>
        )}
      </Stack>
      {busy && <LinearProgress sx={{ mt: 1.5, borderRadius: 2 }} />}
    </Paper>
  )
}
