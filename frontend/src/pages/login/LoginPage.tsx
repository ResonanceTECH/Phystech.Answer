import AdminPanelSettingsOutlinedIcon from '@mui/icons-material/AdminPanelSettingsOutlined'
import BadgeOutlinedIcon from '@mui/icons-material/BadgeOutlined'
import LockOutlinedIcon from '@mui/icons-material/LockOutlined'
import {
  Alert,
  Box,
  Button,
  Card,
  CardContent,
  Divider,
  Stack,
  TextField,
  Typography,
} from '@mui/material'
import { useState, type FormEvent } from 'react'
import { Link, Navigate, useNavigate, useSearchParams } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { DEMO_ACCOUNTS, canManageDocuments, useLogin, useSession } from '@/entities/session'
import { apiErrorMessage, isApiError } from '@/shared/api'
import { config } from '@/shared/config/env'

// Вход по логину и паролю (cookie-сессия на бэкенде). Роль определяет
// доступ: сотрудник — ассистент, документы, онбординг; администратор —
// дополнительно загрузка и контроль документов.
export function LoginPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const { user } = useSession()
  const loginMutation = useLogin()
  const [loginValue, setLoginValue] = useState('')
  const [password, setPassword] = useState('')

  const next = params.get('next')
  const target = (u: { role: string } | null | undefined) =>
    next && next.startsWith('/') ? next : canManageDocuments(u as never) ? paths.admin.root : paths.app.onboarding

  if (user) return <Navigate to={target(user)} replace />

  const submit = (l: string, p: string) =>
    loginMutation.mutate(
      { login: l, password: p },
      { onSuccess: (u) => navigate(target(u), { replace: true }) },
    )

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    submit(loginValue.trim(), password)
  }

  const error = loginMutation.error
  const errorText =
    error && isApiError(error) && error.status === 401 ? 'Неверный логин или пароль.' : error ? apiErrorMessage(error) : ''

  return (
    <Box
      sx={{
        minHeight: '100svh',
        display: 'grid',
        placeItems: 'center',
        p: 2,
        background:
          'radial-gradient(1200px 600px at 80% -10%, rgba(139,92,246,0.18), transparent 60%), radial-gradient(900px 500px at -10% 110%, rgba(109,40,217,0.14), transparent 60%)',
      }}
    >
      <Card elevation={0} sx={{ width: '100%', maxWidth: 440, border: '1px solid', borderColor: 'divider', borderRadius: 4 }}>
        <CardContent sx={{ p: { xs: 3, sm: 4 } }}>
          <Stack spacing={3}>
            <Box>
              <Typography component={Link} to={paths.home} variant="h5" sx={{ fontWeight: 800, color: 'primary.main', textDecoration: 'none' }}>
                {config.appName}
              </Typography>
              <Typography variant="body2" color="text.secondary" sx={{ mt: 0.5 }}>
                Войдите, чтобы задавать вопросы по регламентам и проходить план ознакомления.
              </Typography>
            </Box>

            <Box component="form" onSubmit={onSubmit}>
              <Stack spacing={2}>
                <TextField
                  label="Логин"
                  value={loginValue}
                  onChange={(e) => setLoginValue(e.target.value)}
                  autoComplete="username"
                  autoFocus
                  fullWidth
                />
                <TextField
                  label="Пароль"
                  type="password"
                  value={password}
                  onChange={(e) => setPassword(e.target.value)}
                  autoComplete="current-password"
                  fullWidth
                />
                {errorText && <Alert severity="error">{errorText}</Alert>}
                <Button
                  type="submit"
                  variant="contained"
                  size="large"
                  startIcon={<LockOutlinedIcon />}
                  disabled={!loginValue.trim() || !password || loginMutation.isPending}
                >
                  {loginMutation.isPending ? 'Входим…' : 'Войти'}
                </Button>
              </Stack>
            </Box>

            <Divider>
              <Typography variant="caption" color="text.secondary">
                демо-доступ в один клик
              </Typography>
            </Divider>

            <Stack direction={{ xs: 'column', sm: 'row' }} spacing={1.5}>
              {DEMO_ACCOUNTS.map((acc) => (
                <Button
                  key={acc.login}
                  variant="outlined"
                  fullWidth
                  disabled={loginMutation.isPending}
                  onClick={() => submit(acc.login, acc.password)}
                  startIcon={acc.login === 'admin' ? <AdminPanelSettingsOutlinedIcon /> : <BadgeOutlinedIcon />}
                  sx={{ flexDirection: 'column', alignItems: 'flex-start', py: 1.25, textAlign: 'left' }}
                >
                  <Box>
                    <Typography variant="subtitle2" sx={{ lineHeight: 1.2 }}>
                      {acc.label}
                    </Typography>
                    <Typography variant="caption" color="text.secondary" sx={{ display: 'block', lineHeight: 1.3 }}>
                      {acc.hint}
                    </Typography>
                  </Box>
                </Button>
              ))}
            </Stack>
          </Stack>
        </CardContent>
      </Card>
    </Box>
  )
}
