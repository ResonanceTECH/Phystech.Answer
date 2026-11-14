import { Box, Card, CardActionArea, CardContent, Chip, Typography } from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import type { Document } from '@/entities/document'
import { formatDocDate } from '@/entities/source-reference'
import { buildMatcher, snippets, splitByMatches, type Segment } from '@/shared/lib'

interface DocumentCardProps {
  document: Document
  /** Поисковый запрос: подсвечивается в названии и во фрагментах текста. */
  query?: string
}

function Marked({ segments }: { segments: Segment[] }) {
  return (
    <>
      {segments.map((s, i) =>
        s.hit ? (
          <Box key={i} component="mark" sx={{ bgcolor: '#fde68a', color: 'inherit', borderRadius: '2px', px: '1px' }}>
            {s.text}
          </Box>
        ) : (
          <span key={i}>{s.text}</span>
        ),
      )}
    </>
  )
}

export function DocumentCard({ document: doc, query = '' }: DocumentCardProps) {
  const re = query.trim() ? buildMatcher(query) : null
  const snip = query.trim() ? snippets(doc.content, query, 2, 80) : null
  const requisites = [
    doc.documentType,
    doc.documentNumber && !doc.title.includes(doc.documentNumber) ? `№ ${doc.documentNumber}` : '',
    doc.documentDate ? `от ${formatDocDate(doc.documentDate.slice(0, 10))}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
  const to = paths.app.document(doc.id) + (query.trim() ? `?q=${encodeURIComponent(query.trim())}` : '')

  return (
    <Card
      variant="outlined"
      sx={{ height: '100%', display: 'flex', flexDirection: 'column', transition: 'border-color 0.15s, box-shadow 0.15s', '&:hover': { borderColor: 'primary.main', boxShadow: 1 } }}
    >
      <CardActionArea component={RouterLink} to={to} sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'stretch' }}>
        <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1, alignItems: 'flex-start' }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, lineHeight: 1.35 }}>
              <Marked segments={splitByMatches(doc.title, re)} />
            </Typography>
            {doc.read && <Chip size="small" color="success" variant="outlined" label="ознакомлен" />}
          </Box>
          {requisites && (
            <Typography variant="caption" color="text.secondary">
              {requisites}
            </Typography>
          )}
          {snip && (
            <Typography variant="body2" color="text.secondary" sx={{ lineHeight: 1.55, fontSize: 13 }}>
              <Marked segments={snip.segments} />
            </Typography>
          )}
          <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', mt: 'auto', pt: 0.5, alignItems: 'center' }}>
            {doc.category && <Chip size="small" variant="outlined" label={doc.category} />}
            {snip && <Chip size="small" color="warning" variant="outlined" label={`совпадений: ${snip.total}`} />}
          </Box>
        </CardContent>
      </CardActionArea>
    </Card>
  )
}
