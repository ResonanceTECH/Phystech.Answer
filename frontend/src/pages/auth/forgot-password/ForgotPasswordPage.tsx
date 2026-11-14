import { Link } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { ForgotPasswordForm } from '@/features/auth'
import { AuthShell } from '../AuthShell'

export function ForgotPasswordPage() {
  return (
    <AuthShell
      title="Сброс пароля"
      subtitle="Введите логин — пришлём инструкцию по восстановлению доступа."
    >
      <ForgotPasswordForm loginHref={paths.auth.login} />
      <p className="pl-auth__alt">
        Вспомнили пароль? <Link to={paths.auth.login}>Войти</Link>
      </p>
    </AuthShell>
  )
}
