import { Link, useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { LoginForm } from '@/features/auth'
import { AuthShell } from '../AuthShell'
import { homeFor, useRedirectIfAuthed } from '../lib'

export function LoginPage() {
  const navigate = useNavigate()
  useRedirectIfAuthed()

  return (
    <AuthShell title="Вход" subtitle="Войдите, чтобы продолжить работу с ассистентом.">
      <LoginForm
        forgotPasswordHref={paths.auth.forgotPassword}
        onSuccess={(user) => navigate(homeFor(user), { replace: true })}
      />
      <p className="pl-auth__alt">
        Нет аккаунта? <Link to={paths.auth.register}>Зарегистрироваться</Link>
      </p>
      <p className="pl-auth__hint">
        Демо-доступ — сотрудник <span className="pl-auth__cred">ivan / ivan123</span>,
        администратор <span className="pl-auth__cred">admin / admin123</span>
      </p>
    </AuthShell>
  )
}
