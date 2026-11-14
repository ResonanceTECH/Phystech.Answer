import {
  Box,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  Typography,
} from '@mui/material'
import { Link as RouterLink } from 'react-router-dom'
import type { Document } from '@/entities/document'
import { paths } from '@/app/router/routePaths'
import { formatDate } from '@/shared/lib'

interface DocumentCardProps {
  document: Document
}

export function DocumentCard({ document: doc }: DocumentCardProps) {
  return (
    <Card
      variant="outlined"
      sx={{
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        transition: 'border-color 0.15s, box-shadow 0.15s',
        '&:hover': { borderColor: 'primary.main', boxShadow: 1 },
      }}
    >
      <CardActionArea
        component={RouterLink}
        to={paths.app.document(doc.id)}
        sx={{ flex: 1, display: 'flex', flexDirection: 'column', alignItems: 'stretch' }}
      >
        <CardContent sx={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 1 }}>
          <Box sx={{ display: 'flex', justifyContent: 'space-between', gap: 1, alignItems: 'flex-start' }}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600, lineHeight: 1.35 }}>
              {doc.title}
            </Typography>
            {doc.read && (
              <Chip size="small" color="success" variant="outlined" label="прочитан" />
            )}
          </Box>
          <Box sx={{ display: 'flex', gap: 0.75, flexWrap: 'wrap', flex: 1, alignContent: 'flex-start' }}>
            {doc.tags.map((t) => (
              <Chip key={t} size="small" variant="outlined" label={t} />
            ))}
          </Box>
          <Typography variant="caption" color="text.secondary">
            ред. {doc.version} · обновлён {formatDate(doc.updatedAt)}
          </Typography>
        </CardContent>
      </CardActionArea>
    </Card>
  )
}
