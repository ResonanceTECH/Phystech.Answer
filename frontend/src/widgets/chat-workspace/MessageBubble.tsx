import BookmarkBorderIcon from '@mui/icons-material/BookmarkBorder'
import BookmarkAddedIcon from '@mui/icons-material/BookmarkAdded'
import {
  Alert,
  AlertTitle,
  Box,
  Button,
  Chip,
  Paper,
  Skeleton,
  Stack,
  Tooltip,
  Typography,
} from '@mui/material'
import { useState } from 'react'
import { Link as RouterLink } from 'react-router-dom'
import type { AnswerTrace } from '@/entities/answer'
import type { ChatMessage } from '@/entities/conversation'
import { formatDocDate, type SourceReference } from '@/entities/source-reference'
import { useSaveInstruction } from '@/features/save-instruction'
import { useChatUiStore } from '@/shared/lib'
import { formatTime } from '@/shared/lib'
import { StatusChip, type StatusKind } from '@/shared/ui'

const STATUS_LABEL: Record<string, { kind: StatusKind; label: string }> = {
  answered: { kind: 'success', label: 'Ответ подтверждён источниками' },
  partial: { kind: 'warning', label: 'Частичный ответ' },
  clarification_required: { kind: 'info', label: 'Уточните вопрос' },
  not_found: { kind: 'warning', label: 'Информация не найдена' },
  conflict: { kind: 'warning', label: 'Есть изменения редакций' },
  error: { kind: 'error', label: 'Ошибка' },
}

interface MessageBubbleProps {
  message: ChatMessage
  conversationTitle?: string
}

export function MessageBubble({ message, conversationTitle }: MessageBubbleProps) {
  if (message.role === 'user') {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'flex-end' }}>
        <Paper
          elevation={0}
          sx={{
            maxWidth: '78%',
            px: 2,
            py: 1.25,
            bgcolor: 'primary.main',
            color: 'primary.contrastText',
            borderRadius: 3,
            borderBottomRightRadius: 1,
          }}
        >
          <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.55 }}>
            {message.text}
          </Typography>
          <Typography variant="caption" sx={{ opacity: 0.7, display: 'block', mt: 0.5, textAlign: 'right' }}>
            {formatTime(message.createdAt)}
          </Typography>
        </Paper>
      </Box>
    )
  }

  return <AssistantBubble message={message} conversationTitle={conversationTitle} />
}

function AssistantBubble({
  message,
  conversationTitle,
}: {
  message: ChatMessage
  conversationTitle?: string
}) {
  const openSources = useChatUiStore((s) => s.openSources)
  const saveInstruction = useSaveInstruction()
  const [saved, setSaved] = useState(false)

  if (message.pending) {
    return (
      <Box sx={{ maxWidth: '85%' }}>
        <Paper
          elevation={0}
          sx={{
            px: 2,
            py: 1.5,
            border: '1px solid',
            borderColor: 'divider',
            borderRadius: 3,
            borderBottomLeftRadius: 1,
            bgcolor: 'background.paper',
          }}
        >
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Ищу документы, сверяю редакции и формирую ответ…
          </Typography>
          <Skeleton variant="text" width="90%" />
          <Skeleton variant="text" width="75%" />
          <Skeleton variant="text" width="60%" />
        </Paper>
      </Box>
    )
  }

  const status = message.status ? STATUS_LABEL[message.status] : undefined
  const sources = message.sources ?? []
  const structured = Boolean(message.summary)
  const byRef = new Map(sources.map((s) => [s.ref, s]))
  const openRef = (ref: number) => {
    const src = byRef.get(ref)
    if (src) openSources(src.documentId, src.quote)
  }

  const handleSave = () => {
    saveInstruction({
      title: conversationTitle ?? 'Ответ ассистента',
      answer: message.text,
      sources,
    })
    setSaved(true)
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1, maxWidth: '92%' }}>
      <Paper
        elevation={0}
        sx={{
          px: 2,
          py: 1.5,
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 3,
          borderBottomLeftRadius: 1,
          bgcolor: 'background.paper',
        }}
      >
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1, flexWrap: 'wrap' }}>
          {status && <StatusChip kind={status.kind} label={status.label} />}
          {message.stub && (
            <Chip size="small" variant="outlined" color="warning" label="сервис поиска недоступен" />
          )}
          <Typography variant="caption" color="text.secondary" sx={{ ml: 'auto' }}>
            {formatTime(message.createdAt)}
          </Typography>
        </Box>

        {structured ? (
          <StructuredAnswer message={message} onRef={openRef} />
        ) : (
          <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap', lineHeight: 1.6 }}>
            {message.text}
          </Typography>
        )}

        {sources.length > 0 && (
          <Box sx={{ mt: 2 }}>
            <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700, letterSpacing: 0.4 }}>
              ИСТОЧНИКИ
            </Typography>
            <Stack spacing={0.75} sx={{ mt: 0.75 }}>
              {sources.map((src) => (
                <SourceCard key={src.id} src={src} onOpen={() => openSources(src.documentId, src.quote)} />
              ))}
            </Stack>
          </Box>
        )}

        {message.trace && <TraceLine trace={message.trace} />}

        {!message.stub && message.status !== 'not_found' && message.status !== 'clarification_required' && (
          <Box sx={{ mt: 1 }}>
            <Button
              size="small"
              variant="text"
              startIcon={saved ? <BookmarkAddedIcon /> : <BookmarkBorderIcon />}
              onClick={handleSave}
              disabled={saved}
            >
              {saved ? 'Сохранено' : 'Сохранить инструкцию'}
            </Button>
          </Box>
        )}
      </Paper>
    </Box>
  )
}

// Текст с маркерами [n]: маркеры становятся кликабельными ссылками на источник.
function CitedText({ text, onRef }: { text: string; onRef: (ref: number) => void }) {
  const parts = text.split(/(\[\d{1,2}\])/g)
  return (
    <>
      {parts.map((part, i) => {
        const m = /^\[(\d{1,2})\]$/.exec(part)
        if (!m) return <span key={i}>{part}</span>
        const ref = Number(m[1])
        return (
          <Box
            key={i}
            component="button"
            type="button"
            onClick={() => onRef(ref)}
            sx={{
              border: 0,
              p: 0,
              mx: 0.25,
              px: 0.5,
              borderRadius: 1,
              fontSize: '0.72rem',
              fontWeight: 700,
              lineHeight: 1.5,
              verticalAlign: 'super',
              cursor: 'pointer',
              bgcolor: 'primary.main',
              color: 'primary.contrastText',
              opacity: 0.85,
              '&:hover': { opacity: 1 },
            }}
          >
            {ref}
          </Box>
        )
      })}
    </>
  )
}

function StructuredAnswer({ message, onRef }: { message: ChatMessage; onRef: (ref: number) => void }) {
  const steps = message.steps ?? []
  const notes = message.notes ?? []
  const conflicts = message.conflicts ?? []
  return (
    <Stack spacing={1.5}>
      <Typography variant="body1" sx={{ lineHeight: 1.6, fontWeight: 500 }}>
        <CitedText text={message.summary ?? ''} onRef={onRef} />
      </Typography>

      {steps.length > 0 && (
        <Box>
          <Typography variant="subtitle2" sx={{ mb: 0.75 }}>
            Порядок действий
          </Typography>
          <Stack component="ol" spacing={0.75} sx={{ m: 0, pl: 0, listStyle: 'none', counterReset: 'step' }}>
            {steps.map((step, i) => (
              <Box component="li" key={i} sx={{ display: 'flex', gap: 1.25, alignItems: 'flex-start' }}>
                <Box
                  sx={{
                    flex: '0 0 auto',
                    width: 24,
                    height: 24,
                    borderRadius: '50%',
                    bgcolor: 'action.selected',
                    display: 'grid',
                    placeItems: 'center',
                    fontSize: 12,
                    fontWeight: 700,
                    mt: 0.125,
                  }}
                >
                  {i + 1}
                </Box>
                <Typography variant="body2" sx={{ lineHeight: 1.6 }}>
                  <CitedText text={step.text} onRef={onRef} />
                </Typography>
              </Box>
            ))}
          </Stack>
        </Box>
      )}

      {notes.length > 0 && (
        <Box>
          <Typography variant="subtitle2" sx={{ mb: 0.5 }}>
            Обратите внимание
          </Typography>
          <Box component="ul" sx={{ m: 0, pl: 2.5 }}>
            {notes.map((n, i) => (
              <Typography component="li" variant="body2" key={i} sx={{ lineHeight: 1.6 }}>
                <CitedText text={n} onRef={onRef} />
              </Typography>
            ))}
          </Box>
        </Box>
      )}

      {conflicts.length > 0 && (
        <Alert severity="warning" variant="outlined">
          <AlertTitle sx={{ fontSize: 14 }}>Изменения редакций и противоречия</AlertTitle>
          {conflicts.map((c, i) => (
            <Typography variant="body2" key={i} sx={{ lineHeight: 1.6 }}>
              <CitedText text={c.description} onRef={onRef} />
            </Typography>
          ))}
        </Alert>
      )}

      {message.missing && message.status !== 'answered' && (
        <Alert severity="info" variant="outlined">
          <AlertTitle sx={{ fontSize: 14 }}>
            {message.status === 'not_found' ? 'В базе нет ответа' : 'Не хватает сведений'}
          </AlertTitle>
          <Typography variant="body2">{message.missing}</Typography>
        </Alert>
      )}
    </Stack>
  )
}

// Ссылка в просмотрщик с подсветкой номера пункта («2.5» из «2.5–2.7»).
function sourceLink(src: SourceReference) {
  const num = src.sectionNumber?.match(/\d+(?:\.\d+)*/)?.[0]
  return `/app/documents/${src.documentId}${num ? `?q=${encodeURIComponent(num)}` : ''}`
}

function SourceCard({ src, onOpen }: { src: SourceReference; onOpen: () => void }) {
  const requisites = [
    src.documentNumber && !src.documentTitle.includes(src.documentNumber) ? `№ ${src.documentNumber}` : '',
    src.documentDate ? `от ${formatDocDate(src.documentDate)}` : '',
  ]
    .filter(Boolean)
    .join(' ')
  return (
    <Tooltip title={<Box sx={{ whiteSpace: 'pre-wrap', fontSize: 12 }}>{src.quote}</Box>} arrow placement="top-start">
      <Box
        component="button"
        type="button"
        onClick={onOpen}
        sx={{
          textAlign: 'left',
          display: 'flex',
          gap: 1,
          alignItems: 'flex-start',
          width: '100%',
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 2,
          bgcolor: 'transparent',
          color: 'text.primary',
          px: 1.25,
          py: 0.75,
          cursor: 'pointer',
          font: 'inherit',
          '&:hover': { borderColor: 'primary.main', bgcolor: 'action.hover' },
        }}
      >
        <Box sx={{ fontWeight: 700, fontSize: 12, color: 'primary.main', minWidth: 20 }}>[{src.ref}]</Box>
        <Box sx={{ minWidth: 0 }}>
          <Typography variant="body2" sx={{ fontWeight: 600, lineHeight: 1.4 }}>
            {src.documentTitle}
            {requisites && (
              <Typography component="span" variant="body2" color="text.secondary">
                {' '}
                {requisites}
              </Typography>
            )}
          </Typography>
          <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
            {[src.sectionNumber ? `п. ${src.sectionNumber}` : '', src.heading ?? ''].filter(Boolean).join(' · ')}
          </Typography>
          <Typography
            component={RouterLink}
            to={sourceLink(src)}
            onClick={(e: React.MouseEvent) => e.stopPropagation()}
            variant="caption"
            sx={{ color: 'primary.main', fontWeight: 600 }}
          >
            Открыть в документе →
          </Typography>
        </Box>
      </Box>
    </Tooltip>
  )
}

const MODE_LABEL: Record<string, string> = {
  hybrid: 'гибридный поиск BM25 + векторы',
  bm25: 'лексический поиск BM25',
  vector: 'векторный поиск',
}

function TraceLine({ trace }: { trace: AnswerTrace }) {
  const parts = [
    MODE_LABEL[trace.mode] ?? trace.mode,
    trace.reranked ? 'LLM-реранкинг' : '',
    trace.amendments && trace.amendments.length > 0 ? `учтены изменяющие акты: ${trace.amendments.length}` : '',
    trace.model ? trace.model : '',
    `${(trace.latencyMs / 1000).toFixed(1)} с`,
  ].filter(Boolean)
  return (
    <Tooltip title={trace.amendments?.join('; ') ?? ''} arrow>
      <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 1.25, opacity: 0.8 }}>
        Как найден ответ: {parts.join(' · ')}
      </Typography>
    </Tooltip>
  )
}
