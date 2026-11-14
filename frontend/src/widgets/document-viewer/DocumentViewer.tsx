import KeyboardArrowDownIcon from '@mui/icons-material/KeyboardArrowDown'
import KeyboardArrowUpIcon from '@mui/icons-material/KeyboardArrowUp'
import { Alert, Box, CircularProgress, IconButton, Paper, Stack, Tab, Tabs, Typography } from '@mui/material'
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Document as PdfDocument, Page as PdfPage, pdfjs } from 'react-pdf'
import 'react-pdf/dist/Page/TextLayer.css'
import { config } from '@/shared/config/env'
import { buildMatcher, highlightDom, splitByMatches } from '@/shared/lib'
import './document-viewer.css'

pdfjs.GlobalWorkerOptions.workerSrc = new URL('pdfjs-dist/build/pdf.worker.min.mjs', import.meta.url).toString()

type Kind = 'pdf' | 'docx' | 'xlsx' | 'text' | 'none'

function kindOf(contentType: string): Kind {
  if (contentType.includes('pdf')) return 'pdf'
  if (contentType.includes('wordprocessingml')) return 'docx'
  if (contentType.includes('spreadsheetml')) return 'xlsx'
  if (contentType.startsWith('text/')) return 'text'
  return 'none'
}

interface Props {
  documentId: string
  /** Распознанный текст документа (для вкладки «Текст» и сканов). */
  content: string
  /** Что подсветить (поисковый запрос или фрагмент цитаты). */
  query: string
}

// Просмотр документа «как в оригинале»: PDF — pdf.js (react-pdf), DOCX —
// docx-preview, XLSX — SheetJS. Совпадения запроса подсвечиваются, между
// ними можно переходить. Вкладка «Распознанный текст» — для сканов.
export function DocumentViewer({ documentId, content, query }: Props) {
  const [blob, setBlob] = useState<Blob | null>(null)
  const [kind, setKind] = useState<Kind | null>(null)
  const [loadError, setLoadError] = useState('')
  const [tab, setTab] = useState<'original' | 'text'>('original')

  useEffect(() => {
    let alive = true
    setBlob(null)
    setKind(null)
    setLoadError('')
    fetch(`${config.apiBaseUrl}/v1/documents/${documentId}/file`, { credentials: 'include' })
      .then(async (r) => {
        if (!r.ok) throw new Error(r.status === 404 ? 'Исходный файл недоступен' : `Ошибка ${r.status}`)
        const b = await r.blob()
        if (!alive) return
        setBlob(b)
        setKind(kindOf(r.headers.get('Content-Type') ?? b.type))
      })
      .catch((e: Error) => {
        if (!alive) return
        setLoadError(e.message)
        setKind('none')
        setTab('text')
      })
    return () => {
      alive = false
    }
  }, [documentId])

  return (
    <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 3, overflow: 'hidden' }}>
      <Tabs value={tab} onChange={(_, v) => setTab(v)} sx={{ px: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
        <Tab value="original" label="Оригинал" disabled={kind === 'none'} />
        <Tab value="text" label="Распознанный текст" />
      </Tabs>
      {loadError && tab === 'text' && (
        <Alert severity="info" sx={{ m: 2, mb: 0 }}>
          {loadError} — показан распознанный текст.
        </Alert>
      )}
      {tab === 'original' ? (
        kind === null ? (
          <Box sx={{ p: 6, display: 'grid', placeItems: 'center' }}>
            <CircularProgress />
          </Box>
        ) : kind === 'pdf' && blob ? (
          <PdfView blob={blob} query={query} onNoTextLayer={() => setTab('text')} />
        ) : kind === 'docx' && blob ? (
          <DomView query={query} render={(el) => renderDocx(blob, el)} />
        ) : kind === 'xlsx' && blob ? (
          <XlsxView blob={blob} query={query} />
        ) : (
          <TextView content={content} query={query} />
        )
      ) : (
        <TextView content={content} query={query} />
      )}
    </Paper>
  )
}

// ---------- навигация по совпадениям ----------

function useHitNavigation(container: React.RefObject<HTMLElement | null>, deps: unknown[]) {
  const [count, setCount] = useState(0)
  const [index, setIndex] = useState(0)
  const marks = useRef<HTMLElement[]>([])

  const refresh = useCallback(() => {
    marks.current = container.current ? [...container.current.querySelectorAll<HTMLElement>('mark.doc-hl')] : []
    setCount(marks.current.length)
    setIndex(0)
    marks.current[0]?.classList.add('doc-hl--current')
    marks.current[0]?.scrollIntoView({ block: 'center', behavior: 'smooth' })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps)

  const go = (delta: number) => {
    if (marks.current.length === 0) return
    marks.current[index]?.classList.remove('doc-hl--current')
    const next = (index + delta + marks.current.length) % marks.current.length
    marks.current[next].classList.add('doc-hl--current')
    marks.current[next].scrollIntoView({ block: 'center', behavior: 'smooth' })
    setIndex(next)
  }
  return { count, index, refresh, go }
}

function HitBar({ query, count, index, go }: { query: string; count: number; index: number; go: (d: number) => void }) {
  if (!query.trim()) return null
  return (
    <Stack direction="row" spacing={1} sx={{ alignItems: 'center', px: 2, py: 1, borderBottom: '1px solid', borderColor: 'divider', bgcolor: 'action.hover', position: 'sticky', top: 0, zIndex: 2 }}>
      <Typography variant="body2" sx={{ flex: 1 }}>
        {count > 0 ? (
          <>
            «{query}»: совпадение <b>{index + 1}</b> из <b>{count}</b>
          </>
        ) : (
          <>«{query}»: в этом представлении совпадений нет</>
        )}
      </Typography>
      <IconButton size="small" disabled={count < 2} onClick={() => go(-1)} aria-label="Предыдущее совпадение">
        <KeyboardArrowUpIcon />
      </IconButton>
      <IconButton size="small" disabled={count < 2} onClick={() => go(1)} aria-label="Следующее совпадение">
        <KeyboardArrowDownIcon />
      </IconButton>
    </Stack>
  )
}

// ---------- PDF ----------

function PdfView({ blob, query, onNoTextLayer }: { blob: Blob; query: string; onNoTextLayer: () => void }) {
  const box = useRef<HTMLDivElement>(null)
  const [pages, setPages] = useState(0)
  const [rendered, setRendered] = useState(0)
  const [width, setWidth] = useState(800)
  const [textChars, setTextChars] = useState(0)
  const re = useMemo(() => buildMatcher(query), [query])
  const nav = useHitNavigation(box, [rendered === pages && pages > 0, query])
  const file = useMemo(() => blob, [blob])

  useEffect(() => {
    const el = box.current
    if (!el) return
    const ro = new ResizeObserver(() => setWidth(Math.min(el.clientWidth - 32, 980)))
    ro.observe(el)
    return () => ro.disconnect()
  }, [])
  useEffect(() => {
    if (pages > 0 && rendered === pages) nav.refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [rendered, pages, query])

  // Подсветка внутри текстового слоя pdf.js.
  const renderText = useCallback(
    ({ str }: { str: string }) =>
      splitByMatches(str, re)
        .map((s) => {
          const t = s.text.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c] as string)
          return s.hit ? `<mark class="doc-hl">${t}</mark>` : t
        })
        .join(''),
    [re],
  )

  const isScan = pages > 0 && rendered === pages && textChars < 50 * pages
  return (
    <Box ref={box} sx={{ bgcolor: '#e9e7ef', maxHeight: '78vh', overflow: 'auto' }}>
      <HitBar query={query} count={nav.count} index={nav.index} go={nav.go} />
      {isScan && query.trim() && (
        <Alert
          severity="info"
          sx={{ m: 2 }}
          action={
            <Typography component="button" onClick={onNoTextLayer} sx={{ border: 0, bgcolor: 'transparent', color: 'primary.main', cursor: 'pointer', font: 'inherit', fontWeight: 700 }}>
              Открыть текст
            </Typography>
          }
        >
          Это скан без текстового слоя: подсветка доступна во вкладке «Распознанный текст».
        </Alert>
      )}
      <PdfDocument
        file={file}
        onLoadSuccess={(d) => {
          setPages(d.numPages)
          setRendered(0)
          setTextChars(0)
        }}
        loading={<Box sx={{ p: 6, textAlign: 'center' }}><CircularProgress /></Box>}
        error={<Alert severity="error" sx={{ m: 2 }}>Не удалось открыть PDF.</Alert>}
      >
        {Array.from({ length: pages }, (_, i) => (
          <Box key={i} sx={{ my: 2, mx: 'auto', width: 'fit-content', boxShadow: '0 4px 18px rgba(20,10,40,0.18)', bgcolor: '#fff' }}>
            <PdfPage
              pageNumber={i + 1}
              width={width}
              renderAnnotationLayer={false}
              customTextRenderer={renderText}
              onGetTextSuccess={(t) => setTextChars((c) => c + t.items.reduce((a, it) => a + ('str' in it ? it.str.length : 0), 0))}
              onRenderTextLayerSuccess={() => setRendered((r) => r + 1)}
              onRenderTextLayerError={() => setRendered((r) => r + 1)}
            />
          </Box>
        ))}
      </PdfDocument>
    </Box>
  )
}

// ---------- DOCX / XLSX: рендер в DOM, затем подсветка ----------

async function renderDocx(blob: Blob, el: HTMLElement) {
  const { renderAsync } = await import('docx-preview')
  await renderAsync(blob, el, undefined, {
    className: 'docx',
    inWrapper: true,
    breakPages: true,
    ignoreLastRenderedPageBreak: true,
    experimental: true,
  })
}

function DomView({ query, render }: { query: string; render: (el: HTMLElement) => Promise<void> }) {
  const box = useRef<HTMLDivElement>(null)
  const content = useRef<HTMLDivElement>(null)
  const [ready, setReady] = useState(false)
  const [error, setError] = useState('')
  const nav = useHitNavigation(box, [ready, query])

  useEffect(() => {
    const el = content.current
    if (!el) return
    el.innerHTML = ''
    setReady(false)
    render(el)
      .then(() => setReady(true))
      .catch(() => setError('Не удалось отобразить файл — откройте вкладку «Распознанный текст».'))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [render])

  useEffect(() => {
    if (!ready || !content.current) return
    highlightDom(content.current, query)
    nav.refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ready, query])

  return (
    <Box ref={box} className="doc-dom" sx={{ bgcolor: '#e9e7ef', maxHeight: '78vh', overflow: 'auto' }}>
      <HitBar query={query} count={nav.count} index={nav.index} go={nav.go} />
      {error && <Alert severity="warning" sx={{ m: 2 }}>{error}</Alert>}
      {!ready && !error && (
        <Box sx={{ p: 6, textAlign: 'center' }}>
          <CircularProgress />
        </Box>
      )}
      <div ref={content} />
    </Box>
  )
}

function XlsxView({ blob, query }: { blob: Blob; query: string }) {
  const [sheets, setSheets] = useState<{ name: string; html: string }[]>([])
  const [active, setActive] = useState(0)
  const [error, setError] = useState('')

  useEffect(() => {
    let alive = true
    ;(async () => {
      const XLSX = await import('xlsx')
      const wb = XLSX.read(await blob.arrayBuffer(), { type: 'array' })
      // sheet_to_html экранирует значения ячеек.
      const list = wb.SheetNames.map((name) => ({ name, html: XLSX.utils.sheet_to_html(wb.Sheets[name], { header: '', footer: '' }) }))
      if (alive) setSheets(list)
    })().catch(() => alive && setError('Не удалось открыть таблицу.'))
    return () => {
      alive = false
    }
  }, [blob])

  const render = useCallback(
    async (el: HTMLElement) => {
      el.innerHTML = `<div class="xlsx-sheet">${sheets[active]?.html ?? ''}</div>`
    },
    [sheets, active],
  )

  if (error) return <Alert severity="warning" sx={{ m: 2 }}>{error}</Alert>
  if (sheets.length === 0) return <Box sx={{ p: 6, textAlign: 'center' }}><CircularProgress /></Box>
  return (
    <Box>
      {sheets.length > 1 && (
        <Tabs value={active} onChange={(_, v) => setActive(v)} variant="scrollable" sx={{ px: 1, borderBottom: '1px solid', borderColor: 'divider' }}>
          {sheets.map((s, i) => (
            <Tab key={s.name} value={i} label={s.name} />
          ))}
        </Tabs>
      )}
      <DomView query={query} render={render} />
    </Box>
  )
}

// ---------- распознанный текст ----------

function TextView({ content, query }: { content: string; query: string }) {
  const box = useRef<HTMLDivElement>(null)
  const re = useMemo(() => buildMatcher(query), [query])
  const segs = useMemo(() => splitByMatches(content, re), [content, re])
  const nav = useHitNavigation(box, [content, query])
  useEffect(() => {
    nav.refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [content, query])

  return (
    <Box ref={box} sx={{ maxHeight: '78vh', overflow: 'auto' }}>
      <HitBar query={query} count={nav.count} index={nav.index} go={nav.go} />
      <Box className="doc-text" sx={{ p: { xs: 2, md: 4 }, maxWidth: 860, mx: 'auto' }}>
        {segs.map((s, i) => (s.hit ? <mark key={i} className="doc-hl">{s.text}</mark> : <span key={i}>{s.text}</span>))}
      </Box>
    </Box>
  )
}
