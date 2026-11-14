import ArchiveOutlinedIcon from '@mui/icons-material/ArchiveOutlined'
import AssignmentIndOutlinedIcon from '@mui/icons-material/AssignmentIndOutlined'
import CloudUploadOutlinedIcon from '@mui/icons-material/CloudUploadOutlined'
import DeleteOutlineIcon from '@mui/icons-material/DeleteOutlined'
import DriveFileMoveOutlinedIcon from '@mui/icons-material/DriveFileMoveOutlined'
import FolderOutlinedIcon from '@mui/icons-material/FolderOutlined'
import RestartAltOutlinedIcon from '@mui/icons-material/RestartAltOutlined'
import UnarchiveOutlinedIcon from '@mui/icons-material/UnarchiveOutlined'
import VisibilityOffOutlinedIcon from '@mui/icons-material/VisibilityOffOutlined'
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined'
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Chip,
  List,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Menu,
  MenuItem,
  Paper,
  Stack,
  Tab,
  Tabs,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import { useMemo, useState } from 'react'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { useDocumentsBrief, type Document } from '@/entities/document'
import { formatDocDate } from '@/entities/source-reference'
import {
  FOLDERS,
  VISIBILITY_LABEL,
  useDeleteDocuments,
  usePatchDocuments,
  useReprocessDocument,
  type Visibility,
} from '@/features/admin-kb'
import { AssignDialog } from '@/features/admin-kb/ui/AssignDialog'
import { apiErrorMessage } from '@/shared/api'
import { ConfirmDialog, EmptyState, ErrorState, LoadingState, PageHeader } from '@/shared/ui'

const ALL = '__all__'

function requisites(d: Document) {
  return [
    d.documentType,
    d.documentNumber && !d.title.includes(d.documentNumber) ? `№ ${d.documentNumber}` : '',
    d.documentDate ? `от ${formatDocDate(d.documentDate.slice(0, 10))}` : '',
  ]
    .filter(Boolean)
    .join(' · ')
}

// Управление базой знаний: тематические папки, видимость (скрыть временно,
// архив), удаление, назначение документов для ознакомления.
export function AdminDocumentsPage() {
  const navigate = useNavigate()
  const documents = useDocumentsBrief()
  const patch = usePatchDocuments()
  const remove = useDeleteDocuments()
  const reprocess = useReprocessDocument()

  const [tab, setTab] = useState<Visibility>('active')
  const [folder, setFolder] = useState<string>(ALL)
  const [query, setQuery] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [assignOpen, setAssignOpen] = useState(false)
  const [moveAnchor, setMoveAnchor] = useState<HTMLElement | null>(null)
  const [confirmDelete, setConfirmDelete] = useState(false)

  const all = useMemo(() => documents.data ?? [], [documents.data])
  const vis = (d: Document): Visibility => d.visibility ?? 'active'
  const counts = useMemo(() => {
    const c: Record<Visibility, number> = { active: 0, hidden: 0, archived: 0 }
    all.forEach((d) => (c[vis(d)] += 1))
    return c
  }, [all])
  const inTab = all.filter((d) => vis(d) === tab)
  const folders = useMemo(() => {
    const m = new Map<string, number>()
    inTab.forEach((d) => m.set(d.category || 'Прочие документы', (m.get(d.category || 'Прочие документы') ?? 0) + 1))
    return [...m.entries()].sort(([a], [b]) => a.localeCompare(b))
  }, [inTab])
  const q = query.trim().toLowerCase()
  const shown = inTab
    .filter((d) => folder === ALL || (d.category || 'Прочие документы') === folder)
    .filter((d) => !q || d.title.toLowerCase().includes(q) || (d.documentNumber ?? '').includes(q))
    .sort((a, b) => a.title.localeCompare(b.title))
  const selectedDocs = shown.filter((d) => selected.has(d.id))
  const ids = selectedDocs.map((d) => d.id)

  const toggle = (id: string) =>
    setSelected((s) => {
      const n = new Set(s)
      if (n.has(id)) n.delete(id)
      else n.add(id)
      return n
    })
  const clear = () => setSelected(new Set())
  const setVisibility = (v: Visibility) => patch.mutate({ ids, patch: { visibility: v } }, { onSuccess: clear })

  if (documents.isLoading) return <LoadingState />
  if (documents.isError) return <ErrorState onRetry={() => void documents.refetch()} />

  const error = patch.error ?? remove.error ?? reprocess.error

  return (
    <Box>
      <PageHeader
        title="Документы"
        subtitle="Папки, видимость и назначения. Скрытые и архивные документы не видны сотрудникам и ассистенту."
        actions={
          <Button variant="contained" startIcon={<CloudUploadOutlinedIcon />} onClick={() => navigate(paths.admin.upload)}>
            Загрузить документы
          </Button>
        }
      />

      <Tabs
        value={tab}
        onChange={(_, v: Visibility) => {
          setTab(v)
          setFolder(ALL)
          clear()
        }}
        sx={{ mb: 2 }}
      >
        <Tab value="active" label={`Активные · ${counts.active}`} />
        <Tab value="hidden" label={`Скрытые · ${counts.hidden}`} />
        <Tab value="archived" label={`Архив · ${counts.archived}`} />
      </Tabs>

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={2} sx={{ alignItems: 'flex-start' }}>
        <Paper elevation={0} sx={{ width: { xs: '100%', md: 260 }, flexShrink: 0, border: '1px solid', borderColor: 'divider', borderRadius: 3 }}>
          <List dense>
            <ListItemButton selected={folder === ALL} onClick={() => setFolder(ALL)}>
              <ListItemIcon sx={{ minWidth: 34 }}>
                <FolderOutlinedIcon fontSize="small" />
              </ListItemIcon>
              <ListItemText primary="Все папки" />
              <Chip size="small" label={inTab.length} />
            </ListItemButton>
            {folders.map(([name, n]) => (
              <ListItemButton key={name} selected={folder === name} onClick={() => setFolder(name)}>
                <ListItemIcon sx={{ minWidth: 34 }}>
                  <FolderOutlinedIcon fontSize="small" color="primary" />
                </ListItemIcon>
                <ListItemText primary={name} slotProps={{ primary: { noWrap: true, title: name } }} />
                <Chip size="small" label={n} />
              </ListItemButton>
            ))}
          </List>
        </Paper>

        <Box sx={{ flex: 1, minWidth: 0, width: '100%' }}>
          <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1} sx={{ mb: 1.5, alignItems: { sm: 'center' } }}>
            <TextField size="small" placeholder="Поиск по названию или номеру" value={query} onChange={(e) => setQuery(e.target.value)} sx={{ flex: 1 }} />
          </Stack>

          {ids.length > 0 && (
            <Paper elevation={0} sx={{ p: 1, mb: 1.5, borderRadius: 2, bgcolor: 'rgba(139,92,246,0.08)', border: '1px solid', borderColor: 'primary.light' }}>
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap', gap: 1 }}>
                <Typography variant="body2" sx={{ fontWeight: 700, px: 1 }}>
                  Выбрано: {ids.length}
                </Typography>
                {tab === 'active' && (
                  <Button size="small" startIcon={<AssignmentIndOutlinedIcon />} onClick={() => setAssignOpen(true)}>
                    Назначить
                  </Button>
                )}
                <Button size="small" startIcon={<DriveFileMoveOutlinedIcon />} onClick={(e) => setMoveAnchor(e.currentTarget)}>
                  В папку
                </Button>
                {tab === 'active' && (
                  <Button size="small" startIcon={<VisibilityOffOutlinedIcon />} onClick={() => setVisibility('hidden')}>
                    Скрыть
                  </Button>
                )}
                {tab !== 'active' && (
                  <Button size="small" startIcon={tab === 'archived' ? <UnarchiveOutlinedIcon /> : <VisibilityOutlinedIcon />} onClick={() => setVisibility('active')}>
                    Вернуть в работу
                  </Button>
                )}
                {tab !== 'archived' && (
                  <Button size="small" startIcon={<ArchiveOutlinedIcon />} onClick={() => setVisibility('archived')}>
                    В архив
                  </Button>
                )}
                <Button size="small" color="error" startIcon={<DeleteOutlineIcon />} onClick={() => setConfirmDelete(true)}>
                  Удалить
                </Button>
                <Button size="small" onClick={clear} sx={{ ml: 'auto' }}>
                  Снять выбор
                </Button>
              </Stack>
            </Paper>
          )}

          {error && (
            <Alert severity="error" sx={{ mb: 1.5 }}>
              {apiErrorMessage(error)}
            </Alert>
          )}

          {shown.length === 0 ? (
            <EmptyState
              title={tab === 'active' ? 'Документов нет' : tab === 'hidden' ? 'Скрытых документов нет' : 'Архив пуст'}
              description={tab === 'active' ? 'Загрузите приказы и регламенты — они появятся здесь.' : 'Сюда попадают документы, убранные из работы.'}
            />
          ) : (
            <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 3, overflow: 'hidden' }}>
              <Stack direction="row" sx={{ alignItems: 'center', px: 1, py: 0.5, borderBottom: '1px solid', borderColor: 'divider', bgcolor: 'action.hover' }}>
                <Checkbox
                  size="small"
                  checked={ids.length === shown.length}
                  indeterminate={ids.length > 0 && ids.length < shown.length}
                  onChange={() => (ids.length === shown.length ? clear() : setSelected(new Set(shown.map((d) => d.id))))}
                />
                <Typography variant="caption" color="text.secondary" sx={{ fontWeight: 700 }}>
                  ДОКУМЕНТ
                </Typography>
              </Stack>
              {shown.map((d) => (
                <Stack
                  key={d.id}
                  direction="row"
                  spacing={1}
                  sx={{ alignItems: 'center', px: 1, py: 0.75, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 }, bgcolor: selected.has(d.id) ? 'rgba(139,92,246,0.05)' : undefined }}
                >
                  <Checkbox size="small" checked={selected.has(d.id)} onChange={() => toggle(d.id)} />
                  <Box sx={{ flex: 1, minWidth: 0 }}>
                    <Typography
                      component={RouterLink}
                      to={paths.app.document(d.id)}
                      variant="body2"
                      sx={{ fontWeight: 600, color: 'text.primary', textDecoration: 'none', '&:hover': { color: 'primary.main' } }}
                    >
                      {d.title}
                    </Typography>
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block' }}>
                      {[requisites(d), d.category].filter(Boolean).join(' · ')}
                    </Typography>
                  </Box>
                  {d.processingStatus && d.processingStatus !== 'published' && (
                    <Tooltip title="Статус обработки файла">
                      <Chip size="small" color={d.processingStatus === 'requires_review' ? 'warning' : 'info'} label={d.processingStatus === 'requires_review' ? 'Требует проверки' : 'Обрабатывается'} />
                    </Tooltip>
                  )}
                  {(d.processingStatus === 'requires_review' || d.processingStatus === 'failed') && (
                    <Button
                      size="small"
                      startIcon={<RestartAltOutlinedIcon />}
                      disabled={reprocess.isPending}
                      onClick={() => reprocess.mutate(d.id)}
                    >
                      Обработать снова
                    </Button>
                  )}
                  {tab !== 'active' && <Chip size="small" variant="outlined" label={VISIBILITY_LABEL[vis(d)]} />}
                </Stack>
              ))}
            </Paper>
          )}
        </Box>
      </Stack>

      <Menu anchorEl={moveAnchor} open={Boolean(moveAnchor)} onClose={() => setMoveAnchor(null)}>
        {FOLDERS.map((f) => (
          <MenuItem
            key={f}
            onClick={() => {
              setMoveAnchor(null)
              patch.mutate({ ids, patch: { folder: f } }, { onSuccess: clear })
            }}
          >
            <ListItemIcon>
              <FolderOutlinedIcon fontSize="small" />
            </ListItemIcon>
            {f}
          </MenuItem>
        ))}
      </Menu>

      <AssignDialog open={assignOpen} onClose={() => setAssignOpen(false)} documents={selectedDocs} />

      <ConfirmDialog
        open={confirmDelete}
        title={`Удалить документы (${ids.length})?`}
        description="Документы будут удалены из базы, из ассистента и из планов ознакомления без возможности восстановления. Чтобы убрать документ временно, используйте «Скрыть» или «В архив»."
        confirmLabel="Удалить"
        onCancel={() => setConfirmDelete(false)}
        onConfirm={() => remove.mutate(ids, { onSettled: () => { setConfirmDelete(false); clear() } })}
      />
    </Box>
  )
}
