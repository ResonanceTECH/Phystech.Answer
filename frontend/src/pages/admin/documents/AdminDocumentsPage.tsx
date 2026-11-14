import AddIcon from '@mui/icons-material/Add'
import {
  Alert,
  Box,
  Button,
} from '@mui/material'
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  useDeleteDocument,
  useDocuments,
  type Document,
} from '@/entities/document'
import { DocumentFormDialog } from '@/features/manage-document/ui/DocumentFormDialog'
import { DocumentSearchField } from '@/features/search-documents'
import { paths } from '@/app/router/routePaths'
import { ConfirmDialog, EmptyState, ErrorState, LoadingState, PageHeader } from '@/shared/ui'
import { DocumentsTable } from '@/widgets/documents-table/DocumentsTable'

export function AdminDocumentsPage() {
  const navigate = useNavigate()
  const documents = useDocuments()
  const deleteMutation = useDeleteDocument()

  const [formOpen, setFormOpen] = useState(false)
  const [editing, setEditing] = useState<Document | undefined>()
  const [deleting, setDeleting] = useState<Document | null>(null)

  const filtered = useMemo(() => {
    const docs = documents.data ?? []
    // Страница читает тот же параметр ?q=, что и каталог — единый поиск.
    const q = new URLSearchParams(window.location.search).get('q')?.toLowerCase() ?? ''
    if (!q) return docs
    return docs.filter(
      (d) =>
        d.title.toLowerCase().includes(q) ||
        d.content.toLowerCase().includes(q) ||
        d.tags.some((t) => t.toLowerCase().includes(q)),
    )
  }, [documents.data])

  const confirmDelete = () => {
    if (!deleting) return
    deleteMutation.mutate(deleting.id, { onSettled: () => setDeleting(null) })
  }

  return (
    <Box>
      <PageHeader
        title="Документы"
        subtitle="Управление нормативной базой. Изменения синхронизируются с поисковым индексом ассистента."
        actions={
          <Button
            variant="contained"
            startIcon={<AddIcon />}
            onClick={() => {
              setEditing(undefined)
              setFormOpen(true)
            }}
          >
            Добавить документ
          </Button>
        }
      />

      <DocumentSearchField />

      {deleteMutation.isError && (
        <Alert severity="error" sx={{ mt: 2 }} onClose={() => deleteMutation.reset()}>
          {(deleteMutation.error as Error).message}
        </Alert>
      )}

      {documents.isPending && <LoadingState />}
      {documents.isError && (
        <ErrorState error={documents.error} onRetry={() => documents.refetch()} />
      )}
      {documents.isSuccess && filtered.length === 0 && (
        <EmptyState title="Документы не найдены" />
      )}
      {filtered.length > 0 && (
        <Box sx={{ mt: 2 }}>
          <DocumentsTable
            documents={filtered}
            onView={(doc) => navigate(paths.app.document(doc.id))}
            onEdit={(doc) => {
              setEditing(doc)
              setFormOpen(true)
            }}
            onDelete={(doc) => setDeleting(doc)}
          />
        </Box>
      )}

      <DocumentFormDialog
        open={formOpen}
        document={editing}
        onClose={() => {
          setFormOpen(false)
          setEditing(undefined)
        }}
      />

      <ConfirmDialog
        open={!!deleting}
        title="Удалить документ?"
        description={`«${deleting?.title ?? ''}» будет удалён из базы и поискового индекса. Действие необратимо.`}
        confirmLabel={deleteMutation.isPending ? 'Удаление…' : 'Удалить'}
        danger
        onConfirm={confirmDelete}
        onCancel={() => setDeleting(null)}
      />
    </Box>
  )
}
