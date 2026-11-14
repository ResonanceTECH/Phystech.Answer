import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import ForumOutlinedIcon from '@mui/icons-material/ForumOutlined'
import FolderOutlinedIcon from '@mui/icons-material/FolderOutlined'
import NewReleasesOutlinedIcon from '@mui/icons-material/NewReleasesOutlined'
import StarBorderRoundedIcon from '@mui/icons-material/StarBorderRounded'
import TaskAltIcon from '@mui/icons-material/TaskAlt'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Checkbox,
  Chip,
  CircularProgress,
  LinearProgress,
  Paper,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import { useMemo } from 'react'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { formatDocDate } from '@/entities/source-reference'
import { useAuthStore } from '@/entities/user'
import {
  SOURCE_LABEL,
  dueState,
  groupPlan,
  isNewDocument,
  isUpdatedSinceRead,
  progress,
  useAcknowledge,
  usePlan,
  type PlanDocument,
} from '@/features/reading-plan'
import { ErrorState, LoadingState } from '@/shared/ui'

function requisites(d: PlanDocument) {
  return [
    d.documentNumber && !d.title.includes(d.documentNumber) ? `№ ${d.documentNumber}` : '',
    d.documentDate ? `от ${formatDocDate(d.documentDate.slice(0, 10))}` : '',
  ]
    .filter(Boolean)
    .join(' ')
}

// План ознакомления: обязательные документы (по подразделению, лично, всем)
// со сроками, рекомендуемые для KPI и библиотека по тематическим папкам.
export function ReadingPlanPage() {
  const user = useAuthStore((s) => s.user)
  const { data, isLoading, isError, refetch } = usePlan()
  const groups = useMemo(() => groupPlan(data?.items ?? []), [data])
  const fresh = useMemo(
    () => (data?.items ?? []).filter((d) => d.requirement !== 'library' && (isNewDocument(d) || isUpdatedSinceRead(d))),
    [data],
  )

  if (isLoading) return <LoadingState />
  if (isError) return <ErrorState onRetry={() => void refetch()} />

  const req = progress(groups.required)
  const opt = progress(groups.optional)
  const overdue = groups.required.filter((d) => dueState(d) === 'overdue').length
  const done = req.total > 0 && req.read === req.total

  return (
    <Stack spacing={3} sx={{ maxWidth: 1040, mx: 'auto' }}>
      <Paper
        elevation={0}
        sx={{
          p: { xs: 2.5, md: 3.5 },
          borderRadius: 4,
          color: '#fff',
          background: 'linear-gradient(135deg, #2a1656 0%, #6d28d9 55%, #8b5cf6 100%)',
        }}
      >
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={3} sx={{ alignItems: { sm: 'center' } }}>
          <Box sx={{ position: 'relative', display: 'inline-flex', flexShrink: 0 }}>
            <CircularProgress variant="determinate" value={100} size={112} thickness={4} sx={{ color: 'rgba(255,255,255,0.18)', position: 'absolute' }} />
            <CircularProgress variant="determinate" value={req.pct} size={112} thickness={4} sx={{ color: '#fff' }} />
            <Box sx={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', textAlign: 'center' }}>
              <Box>
                <Typography variant="h4" sx={{ fontWeight: 800, lineHeight: 1 }}>
                  {req.pct}%
                </Typography>
                <Typography variant="caption" sx={{ opacity: 0.85 }}>
                  {req.read} из {req.total}
                </Typography>
              </Box>
            </Box>
          </Box>
          <Box sx={{ flex: 1 }}>
            <Typography variant="h5" sx={{ fontWeight: 800 }}>
              {done ? 'Обязательные документы изучены' : `${user?.firstName || user?.name || ''}, ваш план ознакомления`}
            </Typography>
            <Typography sx={{ opacity: 0.88, mt: 0.75, maxWidth: 600 }}>
              {data?.department ? `Подразделение: ${data.department}. ` : ''}
              Обязательные документы назначены вам, вашему подразделению или всем сотрудникам; рекомендуемые повышают KPI.
            </Typography>
            <Stack direction="row" spacing={1} sx={{ mt: 1.5, flexWrap: 'wrap', gap: 1 }}>
              <Chip label={`Рекомендуемые: ${opt.read}/${opt.total}`} sx={{ bgcolor: 'rgba(255,255,255,0.16)', color: '#fff' }} />
              {overdue > 0 && <Chip color="error" label={`Просрочено: ${overdue}`} />}
            </Stack>
          </Box>
        </Stack>
      </Paper>

      {fresh.length > 0 && (
        <Paper elevation={0} sx={{ p: 2.5, borderRadius: 3, border: '1px solid', borderColor: 'warning.light', bgcolor: 'rgba(245,158,11,0.06)' }}>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1 }}>
            <NewReleasesOutlinedIcon color="warning" />
            <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
              Что нового в ваших регламентах
            </Typography>
          </Stack>
          <Stack spacing={0.75}>
            {fresh.map((d) => (
              <Stack key={d.documentId} direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                <Chip size="small" color={isUpdatedSinceRead(d) ? 'info' : 'warning'} label={isUpdatedSinceRead(d) ? 'Обновлён' : 'Новый'} sx={{ fontWeight: 700 }} />
                <Typography component={RouterLink} to={paths.app.document(d.documentId)} variant="body2" sx={{ fontWeight: 600, color: 'text.primary' }}>
                  {d.title}
                </Typography>
                <Typography variant="caption" color="text.secondary">
                  {requisites(d)}
                </Typography>
              </Stack>
            ))}
          </Stack>
        </Paper>
      )}

      <Section
        title="Обязательные"
        icon={<TaskAltIcon color="primary" />}
        hint="Нужно ознакомиться и поставить отметку. Срок указан, если его задал администратор."
        items={groups.required}
        empty="Обязательных документов пока нет."
      />
      <Section
        title="Рекомендуемые (KPI)"
        icon={<StarBorderRoundedIcon color="secondary" />}
        hint="Не обязательны, но засчитываются в показатели развития."
        items={groups.optional}
        empty="Рекомендаций пока нет."
      />

      {groups.library.size > 0 && (
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
            Библиотека документов
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
            Остальные документы базы по тематическим папкам — по желанию.
          </Typography>
          <Stack spacing={1}>
            {[...groups.library.entries()].map(([folder, docs]) => (
              <Accordion key={folder} disableGutters elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: '12px !important', '&:before': { display: 'none' } }}>
                <AccordionSummary expandIcon={<ExpandMoreIcon />}>
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <FolderOutlinedIcon color="action" />
                    <Typography sx={{ fontWeight: 600 }}>{folder}</Typography>
                    <Chip size="small" label={docs.length} />
                  </Stack>
                </AccordionSummary>
                <AccordionDetails>
                  <Stack spacing={1}>
                    {docs.map((d) => (
                      <DocRow key={d.documentId} doc={d} />
                    ))}
                  </Stack>
                </AccordionDetails>
              </Accordion>
            ))}
          </Stack>
        </Box>
      )}
    </Stack>
  )
}

function Section({ title, icon, hint, items, empty }: { title: string; icon: React.ReactNode; hint: string; items: PlanDocument[]; empty: string }) {
  const p = progress(items)
  return (
    <Box>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
        {icon}
        <Typography variant="h6" sx={{ fontWeight: 700 }}>
          {title}
        </Typography>
        <Typography color="text.secondary" sx={{ fontWeight: 600 }}>
          {p.read}/{p.total}
        </Typography>
      </Stack>
      <Typography variant="body2" color="text.secondary" sx={{ mt: 0.25, mb: 1 }}>
        {hint}
      </Typography>
      {items.length > 0 && <LinearProgress variant="determinate" value={p.pct} sx={{ height: 6, borderRadius: 3, mb: 1.5, maxWidth: 420 }} />}
      {items.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          {empty}
        </Typography>
      ) : (
        <Stack spacing={1}>
          {items.map((d) => (
            <DocRow key={d.documentId} doc={d} />
          ))}
        </Stack>
      )}
    </Box>
  )
}

const DUE_VIEW = {
  overdue: { color: 'error' as const, prefix: 'Просрочено: до' },
  soon: { color: 'warning' as const, prefix: 'Скоро срок: до' },
  ok: { color: 'default' as const, prefix: 'До' },
}

function DocRow({ doc }: { doc: PlanDocument }) {
  const navigate = useNavigate()
  const ack = useAcknowledge()
  const updated = isUpdatedSinceRead(doc)
  const checked = doc.read && !updated
  const due = dueState(doc)
  const askAssistant = () =>
    navigate(paths.app.chatNew, {
      state: {
        question: `Кратко: что важно знать сотруднику из документа «${doc.title}»? Какие действия, сроки и ответственных он устанавливает?`,
      },
    })

  return (
    <Stack
      direction={{ xs: 'column', md: 'row' }}
      spacing={1}
      sx={{
        alignItems: { md: 'center' },
        p: 1,
        pl: 0.5,
        borderRadius: 2,
        border: '1px solid',
        borderColor: due === 'overdue' ? 'error.light' : 'divider',
        bgcolor: checked ? 'action.hover' : 'background.paper',
      }}
    >
      <Stack direction="row" spacing={0.5} sx={{ alignItems: 'center', flex: 1, minWidth: 0 }}>
        <Tooltip title={checked ? `Ознакомлен ${doc.readAt ? new Date(doc.readAt).toLocaleDateString('ru-RU') : ''}` : 'Отметить: ознакомлен'}>
          <span>
            <Checkbox
              checked={checked}
              disabled={checked || ack.isPending}
              onChange={() => ack.mutate(doc.documentId)}
              slotProps={{ input: { 'aria-label': `Ознакомлен: ${doc.title}` } }}
            />
          </span>
        </Tooltip>
        <Box sx={{ minWidth: 0 }}>
          <Typography
            variant="body2"
            component={RouterLink}
            to={paths.app.document(doc.documentId)}
            sx={{ fontWeight: 600, color: 'text.primary', textDecoration: 'none', '&:hover': { color: 'primary.main' } }}
          >
            {doc.title}
          </Typography>
          <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', mt: 0.25, flexWrap: 'wrap', gap: 0.5 }}>
            {requisites(doc) && (
              <Typography variant="caption" color="text.secondary">
                {requisites(doc)}
              </Typography>
            )}
            {doc.source !== 'library' && (
              <Typography variant="caption" color="text.secondary">
                · назначен {SOURCE_LABEL[doc.source]}
              </Typography>
            )}
            {due !== 'none' && doc.dueDate && (
              <Chip size="small" color={DUE_VIEW[due].color} label={`${DUE_VIEW[due].prefix} ${formatDocDate(doc.dueDate.slice(0, 10))}`} sx={{ height: 20, fontWeight: 700 }} />
            )}
            {isNewDocument(doc) && <Chip size="small" color="warning" label="Новый" sx={{ height: 20, fontWeight: 700 }} />}
            {updated && <Chip size="small" color="info" label="Обновлён — ознакомьтесь снова" sx={{ height: 20, fontWeight: 700 }} />}
          </Stack>
        </Box>
      </Stack>
      <Button size="small" startIcon={<ForumOutlinedIcon />} onClick={askAssistant} sx={{ flexShrink: 0, ml: { xs: 5, md: 0 } }}>
        Спросить ассистента
      </Button>
    </Stack>
  )
}
