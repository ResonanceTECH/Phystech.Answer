import DeleteOutlinedIcon from '@mui/icons-material/DeleteOutlined'
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import { Link as RouterLink } from 'react-router-dom'
import { useInstructionStore, type SavedInstruction } from '@/entities/instruction'
import { paths } from '@/app/router/routePaths'
import { formatDateTime } from '@/shared/lib'
import { EmptyState, PageHeader } from '@/shared/ui'

export function InstructionsPage() {
  const instructions = useInstructionStore((s) => s.instructions)
  const remove = useInstructionStore((s) => s.remove)
  const [viewing, setViewing] = useState<SavedInstruction | null>(null)

  return (
    <Box sx={{ maxWidth: 860 }}>
      <PageHeader
        title="Мои инструкции"
        subtitle="Сохранённые ответы ассистента. Хранятся локально в браузере."
      />

      {instructions.length === 0 ? (
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
              <CardContent sx={{ display: 'flex', alignItems: 'center', gap: 2, py: 1.5, '&:last-child': { pb: 1.5 } }}>
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography variant="subtitle2" noWrap>
                    {ins.title}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {formatDateTime(ins.createdAt)} · источников: {ins.sources.length}
                  </Typography>
                </Box>
                <Button size="small" onClick={() => setViewing(ins)}>
                  Открыть
                </Button>
                <Tooltip title="Удалить">
                  <IconButton size="small" color="error" onClick={() => remove(ins.id)} aria-label="Удалить инструкцию">
                    <DeleteOutlinedIcon fontSize="small" />
                  </IconButton>
                </Tooltip>
              </CardContent>
            </Card>
          ))}
        </Box>
      )}

      <Dialog open={!!viewing} onClose={() => setViewing(null)} maxWidth="md" fullWidth>
        <DialogTitle>{viewing?.title}</DialogTitle>
        <DialogContent dividers>
          {viewing && (
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
              <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.65 }}>
                {viewing.answer}
              </Typography>
              {viewing.sources.length > 0 && (
                <Box>
                  <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600 }}>
                    ИСТОЧНИКИ
                  </Typography>
                  <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, mt: 1 }}>
                    {viewing.sources.map((src) => (
                      <Box
                        key={src.id}
                        component={RouterLink}
                        to={paths.app.document(src.documentId)}
                        sx={{ textDecoration: 'none', color: 'inherit' }}
                      >
                        <Card variant="outlined" sx={{ p: 1.5, '&:hover': { borderColor: 'primary.main' } }}>
                          <Typography variant="body2" sx={{ fontWeight: 500 }}>
                            {src.documentTitle}
                            {src.sectionNumber && ` · п. ${src.sectionNumber}`}
                          </Typography>
                          <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 0.5 }}>
                            {src.quote}
                          </Typography>
                        </Card>
                      </Box>
                    ))}
                  </Box>
                </Box>
              )}
              <Box>
                {viewing.sources.map((src) => (
                  <Chip key={src.id} size="small" variant="outlined" label={src.documentTitle} sx={{ mr: 0.5, mb: 0.5 }} />
                ))}
              </Box>
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
