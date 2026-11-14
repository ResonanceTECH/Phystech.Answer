import SaveIcon from '@mui/icons-material/Save'
import {
  Alert,
  Avatar,
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  FormControl,
  Grid,
  InputLabel,
  MenuItem,
  Select,
  Snackbar,
  TextField,
  Typography,
} from '@mui/material'
import { useQuery } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'
import { PageHeader } from '@/shared/ui/PageHeader'
import {
  authApi,
  useAuthStore,
  type EmployeeCategory,
} from '@/entities/user'
import { apiErrorMessage } from '@/shared/api/errors'
import { formatDate } from '@/shared/lib'

const CATEGORIES: { value: EmployeeCategory; label: string }[] = [
  { value: 'employee', label: 'Сотрудник' },
  { value: 'teacher', label: 'Преподаватель' },
  { value: 'researcher', label: 'Исследователь' },
  { value: 'manager', label: 'Руководитель' },
]

const ROLE_LABELS: Record<string, string> = {
  employee: 'Сотрудник',
  admin: 'Администратор',
  expert: 'Эксперт',
}

const CATEGORY_LABELS = Object.fromEntries(
  CATEGORIES.map((c) => [c.value, c.label]),
) as Record<string, string>

export function ProfilePage() {
  const user = useAuthStore((s) => s.user)
  const updateProfile = useAuthStore((s) => s.updateProfile)

  const departments = useQuery({
    queryKey: ['departments'],
    queryFn: authApi.departments,
    staleTime: 5 * 60_000,
  })

  const [firstName, setFirstName] = useState(user?.firstName ?? '')
  const [lastName, setLastName] = useState(user?.lastName ?? '')
  const [category, setCategory] = useState(user?.employeeCategory ?? '')
  const [departmentId, setDepartmentId] = useState(user?.departmentId ?? '')
  const [position, setPosition] = useState(user?.position ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [saved, setSaved] = useState(false)

  if (!user) return null

  const departmentName = (id?: string) =>
    departments.data?.find((d) => d.id === id)?.name

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault()
    setSaving(true)
    setError(null)
    try {
      await updateProfile({
        firstName: firstName.trim() || undefined,
        lastName: lastName.trim() || undefined,
        employeeCategory: (category || undefined) as EmployeeCategory | undefined,
        departmentId: departmentId || undefined,
        position: position.trim() || undefined,
      })
      setSaved(true)
    } catch (err) {
      setError(apiErrorMessage(err))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Box sx={{ maxWidth: 840, mx: 'auto' }}>
      <PageHeader
        title="Личный кабинет"
        subtitle="Данные профиля — ассистент учитывает их при подборе инструкций"
      />

      <Card variant="outlined" sx={{ mb: 3 }}>
        <CardContent sx={{ display: 'flex', gap: 2, alignItems: 'center', flexWrap: 'wrap' }}>
          <Avatar sx={{ width: 56, height: 56, bgcolor: 'primary.main', fontSize: 20 }}>
            {`${user.firstName?.slice(0, 1) ?? ''}${user.lastName?.slice(0, 1) ?? ''}`.toUpperCase() || '•'}
          </Avatar>
          <Box sx={{ flex: 1, minWidth: 200 }}>
            <Typography variant="h6">{user.name}</Typography>
            <Typography variant="body2" color="text.secondary">
              {user.email}
            </Typography>
          </Box>
          <Chip label={ROLE_LABELS[user.role ?? 'employee'] ?? user.role} variant="outlined" />
        </CardContent>
      </Card>

      <Card variant="outlined" component="form" onSubmit={onSubmit}>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Профиль
          </Typography>
          {error ? (
            <Alert severity="error" sx={{ mb: 2 }}>
              {error}
            </Alert>
          ) : null}
          <Grid container spacing={2}>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                label="Имя"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
                fullWidth
                size="small"
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <TextField
                label="Фамилия"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
                fullWidth
                size="small"
              />
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel id="profile-category-label">Категория</InputLabel>
                <Select
                  labelId="profile-category-label"
                  label="Категория"
                  value={category}
                  onChange={(e) => setCategory(e.target.value)}
                >
                  {CATEGORIES.map((c) => (
                    <MenuItem key={c.value} value={c.value}>
                      {c.label}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12, sm: 6 }}>
              <FormControl fullWidth size="small">
                <InputLabel id="profile-department-label">Подразделение</InputLabel>
                <Select
                  labelId="profile-department-label"
                  label="Подразделение"
                  value={departmentId}
                  onChange={(e) => setDepartmentId(e.target.value)}
                  disabled={departments.isLoading}
                >
                  <MenuItem value="">
                    <em>Не выбрано</em>
                  </MenuItem>
                  {(departments.data ?? []).map((d) => (
                    <MenuItem key={d.id} value={d.id}>
                      {d.name}
                    </MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid size={{ xs: 12 }}>
              <TextField
                label="Должность"
                value={position}
                onChange={(e) => setPosition(e.target.value)}
                fullWidth
                size="small"
                placeholder="Например: инженер, доцент, аспирант"
              />
            </Grid>
          </Grid>
          <Box sx={{ mt: 3, display: 'flex', justifyContent: 'flex-end' }}>
            <Button
              type="submit"
              variant="contained"
              disabled={saving}
              startIcon={<SaveIcon />}
            >
              {saving ? 'Сохраняем…' : 'Сохранить изменения'}
            </Button>
          </Box>
        </CardContent>
      </Card>

      <Card variant="outlined" sx={{ mt: 3 }}>
        <CardContent>
          <Typography variant="h6" sx={{ mb: 1 }}>
            Аккаунт
          </Typography>
          <Typography variant="body2" color="text.secondary">
            Зарегистрирован: {formatDate(new Date(user.createdAt))}
            {departmentName(user.departmentId)
              ? ` · ${departmentName(user.departmentId)}`
              : ''}
            {user.employeeCategory && CATEGORY_LABELS[user.employeeCategory]
              ? ` · ${CATEGORY_LABELS[user.employeeCategory]}`
              : ''}
          </Typography>
        </CardContent>
      </Card>

      <Snackbar
        open={saved}
        autoHideDuration={4000}
        onClose={() => setSaved(false)}
        message="Профиль сохранён"
      />
    </Box>
  )
}
