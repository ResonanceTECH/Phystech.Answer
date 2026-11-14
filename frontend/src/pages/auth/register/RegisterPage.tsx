import { Link, useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { RegisterForm } from '@/features/auth'
import { AuthShell } from '../AuthShell'
import { homeFor, useRedirectIfAuthed } from '../lib'

export function RegisterPage() {
  const navigate = useNavigate()
  useRedirectIfAuthed()

  return (
    <AuthShell
      title="Регистрация"
      subtitle="Создайте аккаунт сотрудника, чтобы задавать вопросы ассистенту."
    >
      <RegisterForm onSuccess={(user) => navigate(homeFor(user), { replace: true })} />
      <p className="pl-auth__alt">
        Уже есть аккаунт? <Link to={paths.auth.login}>Войти</Link>
      </p>
      <p className="pl-auth__hint">
        Регистрация создаёт аккаунт сотрудника. Доступ администратора выдаёт
        администратор базы знаний.
      </p>
    </AuthShell>
  )
}
