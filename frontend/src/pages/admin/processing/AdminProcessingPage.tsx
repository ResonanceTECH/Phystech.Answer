import RefreshIcon from '@mui/icons-material/Refresh'
import {
  Box,
  Button,
  Chip,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Typography,
} from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import { api } from '@/shared/api'
import { formatDateTime } from '@/shared/lib'
import { EmptyState, ErrorState, LoadingState, PageHeader } from '@/shared/ui'

interface AdminJob {
  id: string
  documentVersionId?: string
  jobType: string
  status: string
  progress: number
  attempts: number
  maxAttempts: number
  errorCode?: string
  createdAt: string
  completedAt?: string | null
}

const STATUS_COLOR: Record<string, 'default' | 'info' | 'success' | 'warning' | 'error'> = {
  queued: 'info',
  running: 'info',
  pending: 'info',
  succeeded: 'success',
  completed: 'success',
  failed: 'error',
  dead: 'error',
}

const JOB_TYPE_LABEL: Record<string, string> = {
  'document.parse': 'Парсинг документа',
  'answer.generate': 'Генерация ответа',
}

// «Обработка документов» — реальная очередь фоновых задач backend
// (GET /admin/processing, POST /admin/processing/{id}/retry).
export function AdminProcessingPage() {
  const [jobs, setJobs] = useState<AdminJob[] | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [retrying, setRetrying] = useState<string | null>(null)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setJobs(await api.get<AdminJob[]>('/admin/processing?limit=200'))
    } catch {
      setError('Не удалось загрузить очередь обработки.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
    // Автообновление очереди — processing занимает десятки секунд.
    const t = setInterval(() => void load(), 8000)
    return () => clearInterval(t)
  }, [load])

  const retry = async (id: string) => {
    setRetrying(id)
    try {
      await api.post(`/admin/processing/${id}/retry`)
      await load()
    } catch {
      // Оставляем как есть — следующий poll покажет актуальный статус.
    } finally {
      setRetrying(null)
    }
  }

  const retryable = (j: AdminJob) => j.status === 'failed' || j.status === 'dead'

  return (
    <Box>
      <PageHeader
        title="Обработка документов"
        subtitle="Фоновые задачи парсинга и генерации ответов. Обновляется автоматически."
        actions={
          <Button startIcon={<RefreshIcon />} onClick={() => void load()} disabled={loading}>
            Обновить
          </Button>
        }
      />

      {loading && !jobs ? (
        <LoadingState />
      ) : error ? (
        <ErrorState error={new Error(error)} onRetry={() => void load()} />
      ) : !jobs || jobs.length === 0 ? (
        <EmptyState
          title="Очередь пуста"
          description="Задачи появятся после загрузки документов или вопросов ассистенту."
        />
      ) : (
        <TableContainer component={Paper} variant="outlined">
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Задача</TableCell>
                <TableCell>Тип</TableCell>
                <TableCell>Статус</TableCell>
                <TableCell>Попытки</TableCell>
                <TableCell>Создана</TableCell>
                <TableCell>Ошибка</TableCell>
                <TableCell align="right">Действия</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {jobs.map((j) => (
                <TableRow key={j.id} hover>
                  <TableCell>
                    <Typography variant="caption" sx={{ fontFamily: 'monospace' }}>
                      {j.id}
                    </Typography>
                  </TableCell>
                  <TableCell>{JOB_TYPE_LABEL[j.jobType] ?? j.jobType}</TableCell>
                  <TableCell>
                    <Chip
                      size="small"
                      label={j.status}
                      color={STATUS_COLOR[j.status] ?? 'default'}
                      variant="outlined"
                    />
                  </TableCell>
                  <TableCell>
                    {j.attempts}/{j.maxAttempts}
                  </TableCell>
                  <TableCell>{formatDateTime(Date.parse(j.createdAt))}</TableCell>
                  <TableCell>
                    {j.errorCode ? (
                      <Tooltip title={j.errorCode}>
                        <Typography variant="caption" color="error">
                          {j.errorCode}
                        </Typography>
                      </Tooltip>
                    ) : (
                      '—'
                    )}
                  </TableCell>
                  <TableCell align="right">
                    {retryable(j) && (
                      <Button
                        size="small"
                        disabled={retrying === j.id}
                        onClick={() => void retry(j.id)}
                      >
                        Повторить
                      </Button>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </TableContainer>
      )}
    </Box>
  )
}
