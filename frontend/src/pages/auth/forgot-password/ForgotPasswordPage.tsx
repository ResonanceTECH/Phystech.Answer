import { Link } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { ForgotPasswordForm } from '@/features/auth'
import { AuthShell } from '../AuthShell'

export function ForgotPasswordPage() {
  return (
    <AuthShell
      title="Забыли пароль?"
      subtitle="Укажите почту, которую использовали при регистрации. Мы отправим ссылку для создания нового пароля."
    >
      <ForgotPasswordForm loginHref={paths.auth.login} />
      <p className="pl-auth__alt">
        Вспомнили пароль? <Link to={paths.auth.login}>Вернуться ко входу</Link>
      </p>
    </AuthShell>
  )
}
