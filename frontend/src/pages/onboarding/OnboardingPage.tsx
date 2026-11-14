import ArrowForwardIcon from '@mui/icons-material/ArrowForward'
import AutoAwesomeOutlinedIcon from '@mui/icons-material/AutoAwesomeOutlined'
import CheckCircleIcon from '@mui/icons-material/CheckCircle'
import DescriptionOutlinedIcon from '@mui/icons-material/DescriptionOutlined'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import ForumOutlinedIcon from '@mui/icons-material/ForumOutlined'
import NewReleasesOutlinedIcon from '@mui/icons-material/NewReleasesOutlined'
import RadioButtonUncheckedIcon from '@mui/icons-material/RadioButtonUnchecked'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
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
import { useDocumentsBrief, useMarkRead, type Document } from '@/entities/document'
import { useSession } from '@/entities/session'
import { formatDocDate } from '@/entities/source-reference'
import { buildPlan, isNewDocument, isUpdatedSinceRead, type PlannedModule } from '@/features/onboarding'
import { ErrorState, LoadingState } from '@/shared/ui'

function pct(read: number, total: number) {
  return total === 0 ? 0 : Math.round((read / total) * 100)
}

function requisites(d: Document) {
  return [
    d.documentNumber && !d.title.includes(d.documentNumber) ? `№ ${d.documentNumber}` : '',
    d.documentDate ? `от ${formatDocDate(d.documentDate)}` : '',
  ]
    .filter(Boolean)
    .join(' ')
}

// План ознакомления: маршрут по модулям, прогресс, новые и изменённые
// документы, переход к вопросу ассистенту по конкретному документу.
export function OnboardingPage() {
  const { user } = useSession()
  const { data, isLoading, isError, refetch } = useDocumentsBrief()
  const plan = useMemo(() => buildPlan(data ?? []), [data])
  const fresh = useMemo(
    () => (data ?? []).filter((d) => isNewDocument(d) || isUpdatedSinceRead(d)),
    [data],
  )

  if (isLoading) return <LoadingState />
  if (isError) return <ErrorState onRetry={() => void refetch()} />

  const progress = pct(plan.read, plan.total)
  const firstName = user?.name.split(' ')[0] ?? ''
  const done = plan.total > 0 && plan.read === plan.total

  return (
    <Stack spacing={3} sx={{ maxWidth: 1040, mx: 'auto' }}>
      {/* Hero: приветствие и общий прогресс */}
      <Paper
        elevation={0}
        sx={{
          p: { xs: 2.5, md: 3.5 },
          borderRadius: 4,
          color: '#fff',
          background: 'linear-gradient(135deg, #2a1656 0%, #6d28d9 55%, #8b5cf6 100%)',
          position: 'relative',
          overflow: 'hidden',
        }}
      >
        <Box
          sx={{
            position: 'absolute',
            right: -80,
            top: -80,
            width: 280,
            height: 280,
            borderRadius: '50%',
            border: '1px solid rgba(255,255,255,0.18)',
          }}
        />
        <Stack direction={{ xs: 'column', sm: 'row' }} spacing={3} sx={{ alignItems: { sm: 'center' }, position: 'relative' }}>
          <Box sx={{ position: 'relative', display: 'inline-flex', flexShrink: 0 }}>
            <CircularProgress
              variant="determinate"
              value={100}
              size={112}
              thickness={4}
              sx={{ color: 'rgba(255,255,255,0.18)', position: 'absolute' }}
            />
            <CircularProgress variant="determinate" value={progress} size={112} thickness={4} sx={{ color: '#fff' }} />
            <Box sx={{ position: 'absolute', inset: 0, display: 'grid', placeItems: 'center', textAlign: 'center' }}>
              <Box>
                <Typography variant="h4" sx={{ fontWeight: 800, lineHeight: 1 }}>
                  {progress}%
                </Typography>
                <Typography variant="caption" sx={{ opacity: 0.8 }}>
                  {plan.read} из {plan.total}
                </Typography>
              </Box>
            </Box>
          </Box>
          <Box sx={{ flex: 1 }}>
            <Typography variant="h5" sx={{ fontWeight: 800 }}>
              {done ? `${firstName}, план пройден!` : `${firstName ? `${firstName}, д` : 'Д'}обро пожаловать в МФТИ`}
            </Typography>
            <Typography variant="body1" sx={{ opacity: 0.88, mt: 0.75, maxWidth: 560 }}>
              {done
                ? 'Вы познакомились со всеми регламентами базы. Новые и изменённые документы появятся в блоке «Что нового».'
                : 'Ваш маршрут знакомства с внутренними регламентами: по модулю в неделю. Непонятное — спросите ассистента прямо из карточки документа.'}
            </Typography>
            {plan.next && (
              <Button
                component={RouterLink}
                to={paths.app.document(plan.next.id)}
                variant="contained"
                endIcon={<ArrowForwardIcon />}
                sx={{
                  mt: 2,
                  bgcolor: '#fff',
                  color: '#2a1656',
                  fontWeight: 700,
                  '&:hover': { bgcolor: 'rgba(255,255,255,0.9)' },
                }}
              >
                Следующий шаг: {plan.next.title.length > 48 ? `${plan.next.title.slice(0, 46)}…` : plan.next.title}
              </Button>
            )}
          </Box>
        </Stack>
      </Paper>

      {fresh.length > 0 && <WhatsNew docs={fresh} />}

      {/* Маршрут по модулям */}
      <Box>
        <Typography variant="h6" sx={{ fontWeight: 700, mb: 1.5 }}>
          Маршрут ознакомления
        </Typography>
        <Stack spacing={1.5}>
          {plan.modules.map((m, i) => (
            <ModuleCard key={m.id} module={m} index={i} defaultExpanded={m.docs.some((d) => !d.read) && m.id === firstUnfinished(plan.modules)} />
          ))}
          {plan.extra.length > 0 && (
            <ModuleCard
              module={{
                id: 'extra',
                week: 'Дополнительно',
                title: 'Другие документы',
                description: 'Регламенты, которые не вошли в основные модули.',
                keywords: /$^/,
                docs: plan.extra,
                read: plan.extra.filter((d) => d.read).length,
              }}
              index={plan.modules.length}
            />
          )}
        </Stack>
      </Box>

      {plan.reference.length > 0 && (
        <Box>
          <Typography variant="h6" sx={{ fontWeight: 700, mb: 0.5 }}>
            Формы и бланки
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
            Справочные материалы — понадобятся по ходу работы, в прогресс не входят.
          </Typography>
          <Stack direction="row" sx={{ flexWrap: 'wrap', gap: 1 }}>
            {plan.reference.map((d) => (
              <Chip
                key={d.id}
                icon={<DescriptionOutlinedIcon />}
                label={d.title}
                component={RouterLink}
                to={paths.app.document(d.id)}
                clickable
                variant="outlined"
              />
            ))}
          </Stack>
        </Box>
      )}
    </Stack>
  )
}

function firstUnfinished(mods: PlannedModule[]) {
  return mods.find((m) => m.read < m.docs.length)?.id
}

function WhatsNew({ docs }: { docs: Document[] }) {
  return (
    <Paper elevation={0} sx={{ p: 2.5, borderRadius: 3, border: '1px solid', borderColor: 'warning.light', bgcolor: 'rgba(245, 158, 11, 0.06)' }}>
      <Stack direction="row" spacing={1} sx={{ alignItems: 'center', mb: 1.5 }}>
        <NewReleasesOutlinedIcon color="warning" />
        <Typography variant="subtitle1" sx={{ fontWeight: 700 }}>
          Что нового в регламентах
        </Typography>
      </Stack>
      <Stack spacing={1}>
        {docs.map((d) => (
          <Stack key={d.id} direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ alignItems: { sm: 'center' } }}>
            <Chip
              size="small"
              color={isUpdatedSinceRead(d) ? 'info' : 'warning'}
              label={isUpdatedSinceRead(d) ? 'Обновлён' : 'Новый'}
              sx={{ fontWeight: 700, minWidth: 92 }}
            />
            <Typography
              variant="body2"
              component={RouterLink}
              to={paths.app.document(d.id)}
              sx={{ fontWeight: 600, color: 'text.primary', textDecoration: 'none', '&:hover': { color: 'primary.main' } }}
            >
              {d.title}
            </Typography>
            <Typography variant="caption" color="text.secondary">
              {requisites(d)}
            </Typography>
          </Stack>
        ))}
      </Stack>
    </Paper>
  )
}

function ModuleCard({ module: m, index, defaultExpanded }: { module: PlannedModule; index: number; defaultExpanded?: boolean }) {
  const p = pct(m.read, m.docs.length)
  const complete = m.read === m.docs.length
  return (
    <Accordion
      defaultExpanded={defaultExpanded}
      disableGutters
      elevation={0}
      sx={{ border: '1px solid', borderColor: complete ? 'success.light' : 'divider', borderRadius: '16px !important', '&:before': { display: 'none' }, overflow: 'hidden' }}
    >
      <AccordionSummary expandIcon={<ExpandMoreIcon />} sx={{ px: 2.5, py: 1 }}>
        <Stack direction="row" spacing={2} sx={{ alignItems: 'center', width: '100%', pr: 1 }}>
          <Box
            sx={{
              width: 44,
              height: 44,
              flexShrink: 0,
              borderRadius: '50%',
              display: 'grid',
              placeItems: 'center',
              fontWeight: 800,
              color: complete ? 'success.contrastText' : 'primary.main',
              bgcolor: complete ? 'success.main' : 'action.selected',
            }}
          >
            {complete ? <CheckCircleIcon /> : index + 1}
          </Box>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 600, letterSpacing: 0.3 }}>
              {m.week.toUpperCase()}
            </Typography>
            <Typography variant="subtitle1" sx={{ fontWeight: 700, lineHeight: 1.3 }}>
              {m.title}
            </Typography>
            <LinearProgress
              variant="determinate"
              value={p}
              color={complete ? 'success' : 'primary'}
              sx={{ mt: 1, height: 6, borderRadius: 3, maxWidth: 360 }}
            />
          </Box>
          <Typography variant="body2" color="text.secondary" sx={{ flexShrink: 0, fontWeight: 600 }}>
            {m.read}/{m.docs.length}
          </Typography>
        </Stack>
      </AccordionSummary>
      <AccordionDetails sx={{ px: 2.5, pb: 2.5 }}>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 1.5 }}>
          {m.description}
        </Typography>
        <Stack spacing={1}>
          {m.docs.map((d) => (
            <DocRow key={d.id} doc={d} />
          ))}
        </Stack>
      </AccordionDetails>
    </Accordion>
  )
}

function DocRow({ doc }: { doc: Document }) {
  const navigate = useNavigate()
  const markRead = useMarkRead()
  const updated = isUpdatedSinceRead(doc)
  const askAssistant = () =>
    navigate(paths.app.chatNew, {
      state: {
        question: `Кратко: что важно знать сотруднику из документа «${doc.title}»? Какие действия, сроки и ответственные он устанавливает?`,
      },
    })

  return (
    <Stack
      direction={{ xs: 'column', md: 'row' }}
      spacing={1.5}
      sx={{
        alignItems: { md: 'center' },
        p: 1.5,
        borderRadius: 2,
        border: '1px solid',
        borderColor: 'divider',
        bgcolor: doc.read && !updated ? 'action.hover' : 'background.paper',
      }}
    >
      <Stack direction="row" spacing={1.25} sx={{ alignItems: 'center', flex: 1, minWidth: 0 }}>
        {doc.read && !updated ? (
          <CheckCircleIcon color="success" />
        ) : (
          <RadioButtonUncheckedIcon color="disabled" />
        )}
        <Box sx={{ minWidth: 0 }}>
          <Typography
            variant="body2"
            component={RouterLink}
            to={paths.app.document(doc.id)}
            sx={{ fontWeight: 600, color: 'text.primary', textDecoration: 'none', '&:hover': { color: 'primary.main' } }}
          >
            {doc.title}
          </Typography>
          <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center', mt: 0.25, flexWrap: 'wrap' }}>
            {requisites(doc) && (
              <Typography variant="caption" color="text.secondary">
                {requisites(doc)}
              </Typography>
            )}
            {isNewDocument(doc) && <Chip size="small" color="warning" label="Новый" sx={{ height: 20, fontWeight: 700 }} />}
            {updated && <Chip size="small" color="info" label="Обновлён после прочтения" sx={{ height: 20, fontWeight: 700 }} />}
          </Stack>
        </Box>
      </Stack>
      <Stack direction="row" spacing={1} sx={{ flexShrink: 0 }}>
        <Tooltip title="Ассистент кратко перескажет документ со ссылками на пункты">
          <Button size="small" startIcon={<ForumOutlinedIcon />} onClick={askAssistant}>
            Спросить
          </Button>
        </Tooltip>
        {doc.read && !updated ? (
          <Chip size="small" color="success" variant="outlined" icon={<AutoAwesomeOutlinedIcon />} label="Прочитано" />
        ) : (
          <Button
            size="small"
            variant="contained"
            disableElevation
            disabled={markRead.isPending}
            onClick={() => markRead.mutate(doc.id)}
          >
            Прочитано
          </Button>
        )}
      </Stack>
    </Stack>
  )
}
