import { Link, useLocation, useNavigate } from 'react-router-dom'
import { homeFor } from '@/app/router/guards'
import { paths } from '@/app/router/routePaths'
import { LoginForm } from '@/features/auth'
import { config } from '@/shared/config/env'
import { AuthShell } from '../AuthShell'

export function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const state = location.state as { from?: string; reason?: string } | null
  const from =
    state?.from && state.from.startsWith('/') && !state.from.startsWith('//')
      ? state.from
      : null

  return (
    <AuthShell title="С возвращением!" subtitle="Войдите в аккаунт, чтобы продолжить работу с регламентами, инструкциями и документами МФТИ.">
      {state?.reason === 'expired' ? (
        <p className="pl-auth__alert" role="status">
          Сессия истекла — войдите снова.
        </p>
      ) : null}
      <LoginForm
        forgotPasswordHref={paths.auth.forgotPassword}
        onSuccess={(user) =>
          navigate(from ?? (user.onboardingComplete ? homeFor(user) : paths.auth.onboarding), {
            replace: true,
          })
        }
      />
      <p className="pl-auth__alt">
        Нет аккаунта? <Link to={paths.auth.register}>Зарегистрироваться</Link>
      </p>
      {config.demoMode ? (
        <p className="pl-auth__hint">
          Демо-доступ — сотрудник{' '}
          <span className="pl-auth__cred">ivan.petrov@example.com / ivan123</span>, администратор{' '}
          <span className="pl-auth__cred">admin@example.com / admin123</span>
        </p>
      ) : null}
    </AuthShell>
  )
}
