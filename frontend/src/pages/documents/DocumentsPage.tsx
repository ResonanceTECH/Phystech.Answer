import {
  Box,
  Chip,
  Grid,
  Typography,
} from '@mui/material'
import { useMemo } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useDocuments } from '@/entities/document'
import { DocumentSearchField } from '@/features/search-documents'
import { PageHeader, EmptyState, ErrorState, LoadingState } from '@/shared/ui'
import { DocumentCard } from '@/widgets/document-card'

export function DocumentsPage() {
  const [params, setParams] = useSearchParams()
  const query = params.get('q') ?? ''
  const tag = params.get('tag') ?? ''

  const documents = useDocuments()

  const allTags = useMemo(() => {
    const set = new Set<string>()
    for (const d of documents.data ?? []) d.tags.forEach((t) => set.add(t))
    return [...set].sort((a, b) => a.localeCompare(b, 'ru'))
  }, [documents.data])

  const filtered = useMemo(() => {
    const docs = documents.data ?? []
    const q = query.trim().toLowerCase()
    return docs.filter((d) => {
      if (tag && !d.tags.includes(tag)) return false
      if (!q) return true
      return (
        d.title.toLowerCase().includes(q) ||
        d.content.toLowerCase().includes(q) ||
        d.tags.some((t) => t.toLowerCase().includes(q))
      )
    })
  }, [documents.data, query, tag])

  const toggleTag = (t: string) => {
    setParams((prev) => {
      const next = new URLSearchParams(prev)
      if (next.get('tag') === t) next.delete('tag')
      else next.set('tag', t)
      return next
    })
  }

  return (
    <Box sx={{ maxWidth: 1100 }}>
      <PageHeader
        title="Документы"
        subtitle="Нормативная база: регламенты, приказы и инструкции МФТИ."
      />

      <DocumentSearchField />

      {allTags.length > 0 && (
        <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', mt: 1.5, mb: 0.5 }}>
          {allTags.map((t) => (
            <Chip
              key={t}
              size="small"
              variant={tag === t ? 'filled' : 'outlined'}
              color={tag === t ? 'primary' : 'default'}
              label={t}
              onClick={() => toggleTag(t)}
            />
          ))}
        </Box>
      )}

      {documents.isPending && <LoadingState />}
      {documents.isError && (
        <ErrorState error={documents.error} onRetry={() => documents.refetch()} />
      )}
      {documents.isSuccess && filtered.length === 0 && (
        <EmptyState
          title={query || tag ? 'Ничего не найдено' : 'База документов пуста'}
          description={
            query || tag
              ? 'Попробуйте изменить поисковый запрос или сбросить фильтры.'
              : 'Документы появятся после загрузки регламентов.'
          }
        />
      )}

      <Grid container spacing={2} sx={{ mt: 1 }}>
        {filtered.map((doc) => (
          <Grid key={doc.id} size={{ xs: 12, sm: 6, lg: 4 }}>
            <DocumentCard document={doc} />
          </Grid>
        ))}
      </Grid>

      {filtered.length > 0 && (
        <Typography variant="caption" color="text.secondary" sx={{ display: 'block', mt: 2 }}>
          Найдено: {filtered.length}
          {filtered.length !== documents.data?.length &&
            ` из ${documents.data?.length ?? 0}`}
        </Typography>
      )}
    </Box>
  )
}
