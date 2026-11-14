import { Box, Chip, Grid, Typography } from '@mui/material'
import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useDocuments, type Document } from '@/entities/document'
import { DocumentSearchField } from '@/features/search-documents'
import { buildMatcher } from '@/shared/lib'
import { EmptyState, ErrorState, LoadingState, PageHeader } from '@/shared/ui'
import { DocumentCard } from '@/widgets/document-card'

function countHits(re: RegExp, text: string) {
  re.lastIndex = 0
  return (text.toLowerCase().replace(/ё/g, 'е').match(re) ?? []).length
}

// Поиск по базе: формы слов, ранжирование по числу совпадений (название
// весит больше текста), подсветка в карточках, фильтр по папкам.
export function DocumentsPage() {
  const [params, setParams] = useSearchParams()
  const query = params.get('q') ?? ''
  const folder = params.get('folder') ?? ''
  const documents = useDocuments()

  const folders = useMemo(() => {
    const m = new Map<string, number>()
    for (const d of documents.data ?? []) {
      const f = d.category || 'Прочие документы'
      m.set(f, (m.get(f) ?? 0) + 1)
    }
    return [...m.entries()].sort(([a], [b]) => a.localeCompare(b, 'ru'))
  }, [documents.data])

  const results = useMemo(() => {
    const docs = (documents.data ?? []).filter((d) => !folder || (d.category || 'Прочие документы') === folder)
    const re = query.trim() ? buildMatcher(query) : null
    if (!re) return docs.map((d) => ({ d, score: 0 })).sort((a, b) => a.d.title.localeCompare(b.d.title, 'ru'))
    const scored: { d: Document; score: number }[] = []
    for (const d of docs) {
      const score = countHits(re, d.title) * 20 + countHits(re, d.documentNumber ?? '') * 20 + countHits(re, d.content)
      if (score > 0) scored.push({ d, score })
    }
    return scored.sort((a, b) => b.score - a.score)
  }, [documents.data, query, folder])

  const setFolder = (f: string) =>
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      if (!f || next.get('folder') === f) next.delete('folder')
      else next.set('folder', f)
      return next
    })

  return (
    <Box sx={{ maxWidth: 1100 }}>
      <PageHeader title="Документы" subtitle="Нормативная база: регламенты, приказы и формы МФТИ. Поиск учитывает разные формы слов." />
      <DocumentSearchField />

      {folders.length > 0 && (
        <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', mt: 1.5, mb: 0.5 }}>
          {folders.map(([f, n]) => (
            <Chip
              key={f}
              size="small"
              variant={folder === f ? 'filled' : 'outlined'}
              color={folder === f ? 'primary' : 'default'}
              label={`${f} · ${n}`}
              onClick={() => setFolder(f)}
            />
          ))}
        </Box>
      )}

      {documents.isPending && <LoadingState />}
      {documents.isError && <ErrorState error={documents.error} onRetry={() => documents.refetch()} />}
      {documents.isSuccess && results.length === 0 && (
        <EmptyState
          title={query || folder ? 'Ничего не найдено' : 'База документов пуста'}
          description={query || folder ? 'Попробуйте другую формулировку или задайте вопрос ассистенту.' : 'Документы появятся после загрузки регламентов.'}
        />
      )}

      <Grid container spacing={2} sx={{ mt: 1 }}>
        {results.map(({ d }) => (
          <Grid key={d.id} size={{ xs: 12, sm: 6, lg: 4 }}>
            <DocumentCard document={d} query={query} />
          </Grid>
        ))}
      </Grid>

      {results.length > 0 && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 2 }}>
          Найдено: {results.length}
          {results.length !== documents.data?.length && ` из ${documents.data?.length ?? 0}`}
        </Typography>
      )}
    </Box>
  )
}
