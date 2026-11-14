import { useState } from 'react'
import { Link } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { DEMO_ACCOUNTS } from '@/entities/session'
import { authApi, useAuthStore } from '@/entities/user'
import { LoginForm } from '@/features/auth'
import { authErrorMessage } from '@/features/auth/lib/errors'
import { AuthShell } from '../AuthShell'
import { useAfterAuth, useRedirectIfAuthed } from '../lib'

export function LoginPage() {
  const afterAuth = useAfterAuth()
  const setUser = useAuthStore((s) => s.setUser)
  const [demoError, setDemoError] = useState('')
  const [pending, setPending] = useState(false)
  useRedirectIfAuthed()

  // Демо-вход в один клик (учётные записи засеивает бэкенд).
  const demoLogin = async (login: string, password: string) => {
    setPending(true)
    setDemoError('')
    try {
      const user = await authApi.login({ login, password })
      setUser(user)
      afterAuth(user)
    } catch (err) {
      setDemoError(authErrorMessage(err))
    } finally {
      setPending(false)
    }
  }

  return (
    <AuthShell title="Вход" subtitle="Войдите, чтобы продолжить работу с ассистентом.">
      <LoginForm forgotPasswordHref={paths.auth.forgotPassword} onSuccess={afterAuth} />
      <p className="pl-auth__alt">
        Нет аккаунта? <Link to={paths.auth.register}>Зарегистрироваться</Link>
      </p>
      <div className="pl-auth__demo">
        <p className="pl-auth__hint">Демо-доступ в один клик</p>
        <div className="pl-auth__demo-row">
          {DEMO_ACCOUNTS.map((acc) => (
            <button
              key={acc.login}
              type="button"
              className="pl-btn pl-btn--ghost pl-auth__demo-btn"
              disabled={pending}
              onClick={() => void demoLogin(acc.login, acc.password)}
              title={`${acc.login} / ${acc.password}`}
            >
              <span className="pl-auth__demo-label">{acc.label}</span>
              <span className="pl-auth__demo-hint">{acc.hint}</span>
            </button>
          ))}
        </div>
        {demoError ? (
          <p className="pl-auth__alert" role="alert">
            {demoError}
          </p>
        ) : null}
      </div>
    </AuthShell>
  )
}
