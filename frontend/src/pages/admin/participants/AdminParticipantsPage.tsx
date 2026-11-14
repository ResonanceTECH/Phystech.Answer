import AddIcon from '@mui/icons-material/Add'
import AssignmentIndOutlinedIcon from '@mui/icons-material/AssignmentIndOutlined'
import BlockIcon from '@mui/icons-material/Block'
import EditOutlinedIcon from '@mui/icons-material/EditOutlined'
import GroupsOutlinedIcon from '@mui/icons-material/GroupsOutlined'
import LockOpenOutlinedIcon from '@mui/icons-material/LockOpenOutlined'
import {
  Alert,
  Autocomplete,
  Avatar,
  Box,
  Button,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  LinearProgress,
  MenuItem,
  Paper,
  Stack,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import { useEffect, useMemo, useState } from 'react'
import { useDocumentsBrief } from '@/entities/document'
import {
  AssignDialog,
  ROLE_LABEL,
  useCreateDepartment,
  useCreateParticipant,
  useDeleteDepartment,
  useDepartments,
  usePatchParticipant,
  useParticipants,
  type Participant,
} from '@/features/admin-kb'
import { useAuthStore } from '@/entities/user'
import { apiErrorMessage } from '@/shared/api'
import { ConfirmDialog, ErrorState, LoadingState, PageHeader } from '@/shared/ui'

const ALL = '__all__'

function pct(read: number, total: number) {
  return total === 0 ? 100 : Math.round((read / total) * 100)
}

// Панель участников: подразделения, роли, блокировка, прогресс ознакомления
// и персональные назначения документов.
export function AdminParticipantsPage() {
  const me = useAuthStore((s) => s.user)
  const participants = useParticipants()
  const departments = useDepartments()
  const documents = useDocumentsBrief()
  const patch = usePatchParticipant()

  const [dep, setDep] = useState<string>(ALL)
  const [query, setQuery] = useState('')
  const [editing, setEditing] = useState<Participant | null>(null)
  const [assignFor, setAssignFor] = useState<Participant | null>(null)
  const [creating, setCreating] = useState(false)
  const [deptsOpen, setDeptsOpen] = useState(false)

  const list = useMemo(() => {
    const q = query.trim().toLowerCase()
    return (participants.data ?? [])
      .filter((p) => dep === ALL || (dep === '' ? !p.departmentId : p.departmentId === dep))
      .filter((p) => !q || `${p.name} ${p.email} ${p.position ?? ''}`.toLowerCase().includes(q))
  }, [participants.data, dep, query])

  const summary = useMemo(() => {
    const emps = list.filter((p) => p.role === 'employee' && p.active)
    const req = emps.reduce((a, p) => a + p.requiredTotal, 0)
    const read = emps.reduce((a, p) => a + p.requiredRead, 0)
    return { people: emps.length, pct: pct(read, req), done: emps.filter((p) => p.requiredTotal > 0 && p.requiredRead === p.requiredTotal).length }
  }, [list])

  if (participants.isLoading) return <LoadingState />
  if (participants.isError) return <ErrorState onRetry={() => void participants.refetch()} />

  return (
    <Box>
      <PageHeader
        title="Участники"
        subtitle="Сотрудники платформы: подразделения, роли, прогресс ознакомления с обязательными документами."
        actions={
          <Stack direction="row" spacing={1}>
            <Button startIcon={<GroupsOutlinedIcon />} onClick={() => setDeptsOpen(true)}>
              Подразделения
            </Button>
            <Button variant="contained" startIcon={<AddIcon />} onClick={() => setCreating(true)}>
              Добавить сотрудника
            </Button>
          </Stack>
        }
      />

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={1.5} sx={{ mb: 2 }}>
        <TextField select size="small" label="Подразделение" value={dep} onChange={(e) => setDep(e.target.value)} sx={{ minWidth: 300 }}>
          <MenuItem value={ALL}>Все подразделения</MenuItem>
          <MenuItem value="">Без подразделения</MenuItem>
          {(departments.data ?? []).map((d) => (
            <MenuItem key={d.id} value={d.id}>
              {d.name}
            </MenuItem>
          ))}
        </TextField>
        <TextField size="small" placeholder="Поиск по имени, почте, должности" value={query} onChange={(e) => setQuery(e.target.value)} sx={{ flex: 1 }} />
      </Stack>

      <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5} sx={{ mb: 2 }}>
        <Stat label="Сотрудников" value={String(summary.people)} />
        <Stat label="Обязательные изучены" value={`${summary.pct}%`} />
        <Stat label="Прошли план полностью" value={`${summary.done} из ${summary.people}`} />
      </Stack>

      {patch.isError && (
        <Alert severity="error" sx={{ mb: 1.5 }}>
          {apiErrorMessage(patch.error)}
        </Alert>
      )}

      <Paper elevation={0} sx={{ border: '1px solid', borderColor: 'divider', borderRadius: 3, overflow: 'hidden' }}>
        {list.map((p) => (
          <Stack
            key={p.id}
            direction={{ xs: 'column', md: 'row' }}
            spacing={1.5}
            sx={{ alignItems: { md: 'center' }, px: 2, py: 1.25, borderBottom: '1px solid', borderColor: 'divider', '&:last-child': { borderBottom: 0 }, opacity: p.active ? 1 : 0.55 }}
          >
            <Stack direction="row" spacing={1.5} sx={{ alignItems: 'center', flex: 1, minWidth: 0 }}>
              <Avatar sx={{ width: 36, height: 36, bgcolor: p.role === 'employee' ? 'primary.main' : 'secondary.main', fontSize: 14 }}>
                {p.name
                  .split(' ')
                  .map((x) => x[0])
                  .slice(0, 2)
                  .join('')}
              </Avatar>
              <Box sx={{ minWidth: 0 }}>
                <Stack direction="row" spacing={0.75} sx={{ alignItems: 'center' }}>
                  <Typography variant="body2" sx={{ fontWeight: 700 }} noWrap>
                    {p.name}
                  </Typography>
                  {p.role !== 'employee' && <Chip size="small" label={ROLE_LABEL[p.role]} sx={{ height: 20 }} />}
                  {!p.active && <Chip size="small" color="error" label="Заблокирован" sx={{ height: 20 }} />}
                </Stack>
                <Typography variant="caption" color="text.secondary" noWrap sx={{ display: 'block' }}>
                  {[p.position, p.departmentName, p.email].filter(Boolean).join(' · ')}
                </Typography>
              </Box>
            </Stack>

            {p.role === 'employee' && (
              <Box sx={{ width: { xs: '100%', md: 260 } }}>
                <Stack direction="row" sx={{ justifyContent: 'space-between' }}>
                  <Typography variant="caption" color="text.secondary">
                    Обязательные {p.requiredRead}/{p.requiredTotal}
                  </Typography>
                  <Typography variant="caption" color="text.secondary">
                    KPI {p.optionalRead}/{p.optionalTotal}
                  </Typography>
                </Stack>
                <LinearProgress
                  variant="determinate"
                  value={pct(p.requiredRead, p.requiredTotal)}
                  color={p.requiredTotal > 0 && p.requiredRead === p.requiredTotal ? 'success' : 'primary'}
                  sx={{ height: 6, borderRadius: 3, mt: 0.5 }}
                />
              </Box>
            )}

            <Stack direction="row" spacing={0.5} sx={{ flexShrink: 0 }}>
              {p.role === 'employee' && p.active && (
                <Tooltip title="Назначить документы лично">
                  <IconButton onClick={() => setAssignFor(p)}>
                    <AssignmentIndOutlinedIcon />
                  </IconButton>
                </Tooltip>
              )}
              <Tooltip title="Изменить подразделение, должность, роль">
                <IconButton onClick={() => setEditing(p)}>
                  <EditOutlinedIcon />
                </IconButton>
              </Tooltip>
              {p.id !== me?.id && (
                <Tooltip title={p.active ? 'Заблокировать доступ' : 'Разблокировать'}>
                  <IconButton onClick={() => patch.mutate({ id: p.id, patch: { active: !p.active } })}>
                    {p.active ? <BlockIcon /> : <LockOpenOutlinedIcon />}
                  </IconButton>
                </Tooltip>
              )}
            </Stack>
          </Stack>
        ))}
        {list.length === 0 && (
          <Typography color="text.secondary" sx={{ p: 3, textAlign: 'center' }}>
            Никого не найдено.
          </Typography>
        )}
      </Paper>

      <EditParticipantDialog participant={editing} onClose={() => setEditing(null)} isSelf={editing?.id === me?.id} />
      <CreateParticipantDialog open={creating} onClose={() => setCreating(false)} />
      <DepartmentsDialog open={deptsOpen} onClose={() => setDeptsOpen(false)} />
      <AssignDialog
        open={Boolean(assignFor)}
        onClose={() => setAssignFor(null)}
        allDocuments={documents.data ?? []}
        target={assignFor ? { type: 'user', id: assignFor.id } : undefined}
      />
    </Box>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <Paper elevation={0} sx={{ flex: 1, p: 2, border: '1px solid', borderColor: 'divider', borderRadius: 3 }}>
      <Typography variant="caption" color="text.secondary">
        {label}
      </Typography>
      <Typography variant="h5" sx={{ fontWeight: 800 }}>
        {value}
      </Typography>
    </Paper>
  )
}

function EditParticipantDialog({ participant, onClose, isSelf }: { participant: Participant | null; onClose: () => void; isSelf: boolean }) {
  const departments = useDepartments()
  const patch = usePatchParticipant()
  const [role, setRole] = useState<Participant['role']>('employee')
  const [departmentId, setDepartmentId] = useState('')
  const [position, setPosition] = useState('')

  useEffect(() => {
    if (!participant) return
    setRole(participant.role)
    setDepartmentId(participant.departmentId ?? '')
    setPosition(participant.position ?? '')
    patch.reset()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [participant])

  return (
    <Dialog open={Boolean(participant)} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>{participant?.name}</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <Autocomplete
            options={departments.data ?? []}
            getOptionLabel={(d) => d.name}
            value={(departments.data ?? []).find((d) => d.id === departmentId) ?? null}
            onChange={(_, v) => setDepartmentId(v?.id ?? '')}
            renderInput={(params) => <TextField {...params} label="Подразделение" helperText="Документы подразделения назначаются автоматически" />}
          />
          <TextField label="Должность" value={position} onChange={(e) => setPosition(e.target.value)} />
          <TextField select label="Роль" value={role} disabled={isSelf} onChange={(e) => setRole(e.target.value as Participant['role'])}>
            {(['employee', 'expert', 'admin'] as const).map((r) => (
              <MenuItem key={r} value={r}>
                {ROLE_LABEL[r]}
              </MenuItem>
            ))}
          </TextField>
          {patch.isError && <Alert severity="error">{apiErrorMessage(patch.error)}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Отмена</Button>
        <Button
          variant="contained"
          disabled={patch.isPending}
          onClick={() =>
            participant &&
            patch.mutate(
              { id: participant.id, patch: { departmentId, position, ...(isSelf ? {} : { role }) } },
              { onSuccess: onClose },
            )
          }
        >
          Сохранить
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function CreateParticipantDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const departments = useDepartments()
  const create = useCreateParticipant()
  const [form, setForm] = useState({ name: '', email: '', login: '', password: '', position: '', departmentId: '' })
  useEffect(() => {
    if (open) {
      setForm({ name: '', email: '', login: '', password: '', position: '', departmentId: '' })
      create.reset()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])
  const set = (k: keyof typeof form) => (e: React.ChangeEvent<HTMLInputElement>) => setForm({ ...form, [k]: e.target.value })
  const valid = form.name.trim() && (form.login.trim() || form.email.trim()) && form.password.length >= 6

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Новый сотрудник</DialogTitle>
      <DialogContent>
        <Stack spacing={2} sx={{ pt: 1 }}>
          <TextField label="ФИО" value={form.name} onChange={set('name')} autoFocus />
          <TextField label="Электронная почта" type="email" value={form.email} onChange={set('email')} />
          <TextField label="Логин (необязательно)" value={form.login} onChange={set('login')} />
          <TextField label="Временный пароль" type="password" value={form.password} onChange={set('password')} helperText="Не короче 6 символов" />
          <TextField label="Должность" value={form.position} onChange={set('position')} />
          <Autocomplete
            options={departments.data ?? []}
            getOptionLabel={(d) => d.name}
            value={(departments.data ?? []).find((d) => d.id === form.departmentId) ?? null}
            onChange={(_, v) => setForm({ ...form, departmentId: v?.id ?? '' })}
            renderInput={(params) => <TextField {...params} label="Подразделение" />}
          />
          {create.isError && <Alert severity="error">{apiErrorMessage(create.error)}</Alert>}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Отмена</Button>
        <Button
          variant="contained"
          disabled={!valid || create.isPending}
          onClick={() =>
            create.mutate(
              {
                name: form.name.trim(),
                email: form.email.trim() || undefined,
                login: form.login.trim() || form.email.trim(),
                password: form.password,
                position: form.position.trim() || undefined,
                departmentId: form.departmentId || undefined,
              },
              { onSuccess: onClose },
            )
          }
        >
          Добавить
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function DepartmentsDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const departments = useDepartments()
  const create = useCreateDepartment()
  const del = useDeleteDepartment()
  const [name, setName] = useState('')
  const [deleting, setDeleting] = useState<{ id: string; name: string } | null>(null)

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Подразделения</DialogTitle>
      <DialogContent>
        <Stack direction="row" spacing={1} sx={{ my: 1 }}>
          <TextField size="small" fullWidth placeholder="Новое подразделение" value={name} onChange={(e) => setName(e.target.value)} />
          <Button variant="contained" disabled={!name.trim() || create.isPending} onClick={() => create.mutate(name.trim(), { onSuccess: () => setName('') })}>
            Добавить
          </Button>
        </Stack>
        {create.isError && <Alert severity="error">{apiErrorMessage(create.error)}</Alert>}
        <Stack sx={{ mt: 1 }}>
          {(departments.data ?? []).map((d) => (
            <Stack key={d.id} direction="row" sx={{ alignItems: 'center', py: 0.75, borderBottom: '1px solid', borderColor: 'divider' }}>
              <Typography variant="body2" sx={{ flex: 1 }}>
                {d.name}
              </Typography>
              <Chip size="small" label={`${d.members ?? 0} чел.`} sx={{ mr: 1 }} />
              <Button size="small" color="error" onClick={() => setDeleting(d)}>
                Удалить
              </Button>
            </Stack>
          ))}
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Готово</Button>
      </DialogActions>
      <ConfirmDialog
        open={Boolean(deleting)}
        title={`Удалить «${deleting?.name}»?`}
        description="Сотрудники останутся без подразделения, назначения этого подразделения будут сняты."
        confirmLabel="Удалить"
        onCancel={() => setDeleting(null)}
        onConfirm={() => deleting && del.mutate(deleting.id, { onSettled: () => setDeleting(null) })}
      />
    </Dialog>
  )
}
