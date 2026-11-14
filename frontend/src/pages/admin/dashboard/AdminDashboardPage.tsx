import {
  Card,
  CardActionArea,
  CardContent,
  Grid,
  Typography,
} from '@mui/material'
import { useQuery } from '@tanstack/react-query'
import { Link as RouterLink } from 'react-router-dom'
import { useDocuments } from '@/entities/document'
import { paths } from '@/app/router/routePaths'
import { api, queryKeys } from '@/shared/api'
import { PageHeader } from '@/shared/ui'

interface UserSummary {
  id: string
  name: string
}

export function AdminDashboardPage() {
  const documents = useDocuments()
  const users = useQuery({
    queryKey: queryKeys.users.list(),
    queryFn: () => api.get<UserSummary[]>('/v1/users'),
  })

  const docs = documents.data ?? []
  const tagCount = new Set(docs.flatMap((d) => d.tags)).size
  const recent = [...docs]
    .sort((a, b) => b.updatedAt.localeCompare(a.updatedAt))
    .slice(0, 5)

  const stats = [
    { label: 'Документов в базе', value: docs.length },
    { label: 'Пользователей', value: users.data?.length ?? '—' },
    { label: 'Уникальных тегов', value: tagCount },
  ]

  return (
    <>
      <PageHeader
        title="Обзор"
        subtitle="Состояние нормативной базы и индекса ассистента."
      />

      <Grid container spacing={2} sx={{ mb: 4 }}>
        {stats.map((s) => (
          <Grid key={s.label} size={{ xs: 12, sm: 4 }}>
            <Card variant="outlined">
              <CardContent>
                <Typography variant="h4" sx={{ fontWeight: 650 }}>
                  {s.value}
                </Typography>
                <Typography variant="body2" color="text.secondary">
                  {s.label}
                </Typography>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      <Typography variant="h6" sx={{ mb: 1.5 }}>
        Последние обновлённые документы
      </Typography>
      {recent.length === 0 ? (
        <Typography variant="body2" color="text.secondary">
          Документов пока нет.
        </Typography>
      ) : (
        <Grid container spacing={1.5}>
          {recent.map((d) => (
            <Grid key={d.id} size={{ xs: 12, sm: 6 }}>
              <Card variant="outlined">
                <CardActionArea
                  component={RouterLink}
                  to={paths.app.document(d.id)}
                  sx={{ px: 2, py: 1.5 }}
                >
                  <Typography variant="body2" noWrap sx={{ fontWeight: 500 }}>
                    {d.title}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    {d.tags.join(' · ') || 'без тегов'}
                  </Typography>
                </CardActionArea>
              </Card>
            </Grid>
          ))}
        </Grid>
      )}
    </>
  )
}
