import { Link, useNavigate } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { RegisterForm } from '@/features/auth'
import { AuthShell } from '../AuthShell'

export function RegisterPage() {
  const navigate = useNavigate()

  return (
    <AuthShell
      title="Начните работу"
      subtitle="Создайте аккаунт, чтобы получать инструкции по внутренним процедурам МФТИ и сохранять результаты работы."
    >
      <RegisterForm
        onSuccess={() => navigate(paths.auth.onboarding, { replace: true })}
      />
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
