import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined'
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  Card,
  CardContent,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material'
import { useCallback, useEffect, useState } from 'react'
import { Link as RouterLink } from 'react-router-dom'
import {
  conversationApi,
  savedInstructionApi,
  toAnswerFields,
  toSourceReference,
  type ServerAnswer,
  type ServerSavedInstruction,
} from '@/entities/conversation'
import { paths } from '@/app/router/routePaths'
import { formatDateTime } from '@/shared/lib'
import { EmptyState, PageHeader, StatusChip, type StatusKind } from '@/shared/ui'

const STATUS_LABEL: Record<string, { kind: StatusKind; label: string }> = {
  answered: { kind: 'success', label: 'Ответ подтверждён источниками' },
  partial: { kind: 'warning', label: 'Частичный ответ' },
  clarification_required: { kind: 'info', label: 'Нужен уточняющий вопрос' },
  not_found: { kind: 'warning', label: 'Информация не найдена' },
  conflict: { kind: 'warning', label: 'Противоречия в документах' },
  error: { kind: 'error', label: 'Ошибка' },
}

// «Мои инструкции» — серверные сохранённые ответы ассистента
// (/saved-instructions + /answers/{id}).
export function InstructionsPage() {
  const [instructions, setInstructions] = useState<ServerSavedInstruction[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [viewing, setViewing] = useState<ServerSavedInstruction | null>(null)
  const [detail, setDetail] = useState<ServerAnswer | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      setInstructions(await savedInstructionApi.list())
    } catch {
      setError('Не удалось загрузить инструкции. Проверьте соединение.')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const openDetail = async (ins: ServerSavedInstruction) => {
    setViewing(ins)
    setDetail(null)
    setDetailLoading(true)
    try {
      setDetail(await conversationApi.getAnswer(ins.answerId))
    } catch {
      setDetail(null)
    } finally {
      setDetailLoading(false)
    }
  }

  const remove = async (id: string) => {
    try {
      await savedInstructionApi.remove(id)
      setInstructions((list) => list.filter((i) => i.id !== id))
      if (viewing?.id === id) setViewing(null)
    } catch {
      // Оставляем элемент — пользователь может повторить.
    }
  }

  const sources = (detail?.sources ?? []).map(toSourceReference)
  const fields = detail ? toAnswerFields(detail) : null
  const status = detail?.status ? STATUS_LABEL[detail.status] : undefined

  return (
    <Box sx={{ maxWidth: 860 }}>
      <PageHeader
        title="Мои инструкции"
        subtitle="Проверенные ответы ассистента, сохранённые на сервере."
      />

      {loading ? (
        <Typography variant="body2" color="text.secondary">
          Загрузка…
        </Typography>
      ) : error ? (
        <EmptyState
          title="Не удалось загрузить инструкции"
          description={error}
          action={
            <Button variant="contained" onClick={() => void load()}>
              Повторить
            </Button>
          }
        />
      ) : instructions.length === 0 ? (
        <EmptyState
          title="Пока нет сохранённых инструкций"
          description="Откройте ответ ассистента и нажмите «Сохранить инструкцию», чтобы вернуться к нему позже."
          action={
            <Button component={RouterLink} to={paths.app.chatNew} variant="contained">
              Задать вопрос
            </Button>
          }
        />
      ) : (
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
          {instructions.map((ins) => (
            <Card key={ins.id} variant="outlined">
              <CardContent
                sx={{
                  display: 'flex',
                  alignItems: 'center',
                  gap: 2,
                  py: 1.5,
                  '&:last-child': { pb: 1.5 },
                }}
              >
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography variant="subtitle2" noWrap>
                    {ins.title}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {formatDateTime(Date.parse(ins.createdAt))}
                  </Typography>
                </Box>
                <Button size="small" onClick={() => void openDetail(ins)}>
                  Открыть
                </Button>
                <Tooltip title="Удалить">
                  <IconButton
                    size="small"
                    color="error"
                    onClick={() => void remove(ins.id)}
                    aria-label="Удалить инструкцию"
                  >
                    <DeleteOutlinedIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </CardContent>
            </Card>
          ))}
        </Box>
      )}

      <Dialog
        open={!!viewing}
        onClose={() => setViewing(null)}
        maxWidth="md"
        fullWidth
      >
        <DialogTitle>{viewing?.title}</DialogTitle>
        <DialogContent dividers>
          {detailLoading && (
            <Typography variant="body2" color="text.secondary">
              Загрузка ответа…
            </Typography>
          )}
          {!detailLoading && !detail && (
            <Typography variant="body2" color="text.secondary">
              Не удалось загрузить содержимое инструкции.
            </Typography>
          )}
          {detail && fields && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              {status && <StatusChip kind={status.kind} label={status.label} />}
              <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.65 }}>
                {fields.summary}
              </Typography>

              {fields.steps.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
                    Порядок действий
                  </Typography>
                  <Box component="ol" sx={{ m: 0, pl: 2.5 }}>
                    {fields.steps.map((step, i) => (
                      <li key={i}>
                        <Typography variant="body2" sx={{ lineHeight: 1.6 }}>
                          {step.text}
                        </Typography>
                      </li>
                    ))}
                  </Box>
                </Box>
              )}

              {fields.notes.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
                    Обратите внимание
                  </Typography>
                  <Box component="ul" sx={{ m: 0, pl: 2.5 }}>
                    {fields.notes.map((n, i) => (
                      <Typography component="li" variant="body2" key={i} sx={{ lineHeight: 1.6 }}>
                        {n}
                      </Typography>
                    ))}
                  </Box>
                </Box>
              )}

              {fields.conflicts.length > 0 && (
                <Alert severity="warning" variant="outlined">
                  <AlertTitle sx={{ fontSize: 14 }}>Изменения редакций и противоречия</AlertTitle>
                  {fields.conflicts.map((c, i) => (
                    <Typography variant="body2" key={i} sx={{ lineHeight: 1.6 }}>
                      {c.description}
                    </Typography>
                  ))}
                </Alert>
              )}

              {fields.missing && detail.status !== 'answered' && (
                <Alert severity="info" variant="outlined">
                  <AlertTitle sx={{ fontSize: 14 }}>
                    {detail.status === 'not_found' ? 'В базе нет ответа' : 'Не хватает сведений'}
                  </AlertTitle>
                  <Typography variant="body2">{fields.missing}</Typography>
                </Alert>
              )}

              {sources.length > 0 && (
                <Box>
                  <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
                    Источники
                  </Typography>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                    {sources.map((src) =>
                      src.documentId ? (
                        <Box
                          key={src.id}
                          component={RouterLink}
                          to={paths.app.document(src.documentId)}
                          sx={{ textDecoration: 'none', color: 'inherit' }}
                        >
                          <Card
                            variant="outlined"
                            sx={{ p: 1.5, '&:hover': { borderColor: 'primary.main' } }}
                          >
                            <Typography variant="body2" sx={{ fontWeight: 500 }}>
                              {src.documentTitle}
                              {src.sectionNumber && ` · п. ${src.sectionNumber}`}
                            </Typography>
                            <Typography
                              variant="caption"
                              color="text.secondary"
                              sx={{ display: 'block', mt: 0.5 }}
                            >
                              {src.quote}
                            </Typography>
                          </Card>
                        </Box>
                      ) : (
                        <Card key={src.id} variant="outlined" sx={{ p: 1.5 }}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {src.documentTitle}
                            {src.sectionNumber && ` · п. ${src.sectionNumber}`}
                          </Typography>
                          <Typography
                            variant="caption"
                            color="text.secondary"
                            sx={{ display: 'block', mt: 0.5 }}
                          >
                            {src.quote}
                          </Typography>
                        </Card>
                      ),
                    )}
                  </Box>
                </Box>
              )}
            </Box>
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setViewing(null)}>Закрыть</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
