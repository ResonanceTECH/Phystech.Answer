import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { ResetPasswordForm, type ResetResult } from '@/features/auth'
import { AuthShell } from '../AuthShell'

// Токен считывается из ?token= один раз и сразу убирается из адресной
// строки — в URL, истории и реферерах он не остаётся.
export function ResetPasswordPage() {
  const [params] = useSearchParams()
  const [token] = useState(() => params.get('token') ?? '')
  const [result, setResult] = useState<ResetResult | null>(null)

  useEffect(() => {
    const meta = document.createElement('meta')
    meta.name = 'referrer'
    meta.content = 'no-referrer'
    document.head.appendChild(meta)

    if (params.get('token')) {
      const url = new URL(window.location.href)
      url.searchParams.delete('token')
      window.history.replaceState(null, '', url.pathname + url.search)
    }
    return () => {
      document.head.removeChild(meta)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  if (!token) {
    return (
      <AuthShell
        title="Ссылка недействительна"
        subtitle="В ссылке восстановления нет токена. Запросите новую ссылку."
      >
        <ResetFail forgotHref={paths.auth.forgotPassword} />
      </AuthShell>
    )
  }

  if (result?.kind === 'success') {
    return (
      <AuthShell title="Пароль изменён" subtitle="Теперь вы можете войти в аккаунт с новым паролем.">
        <div className="pl-auth__done">
          <div className="pl-auth__icon" aria-hidden>
            <svg viewBox="0 0 24 24" width="26" height="26" fill="none">
              <path
                d="m4.5 12.5 5 5 10-11"
                stroke="currentColor"
                strokeWidth="1.8"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>
          <Link to={paths.auth.login} className="pl-btn pl-btn--primary pl-auth__submit">
            Перейти ко входу
          </Link>
        </div>
      </AuthShell>
    )
  }

  if (result) {
    return (
      <AuthShell title="Ссылка недействительна" subtitle={result.message}>
        <ResetFail forgotHref={paths.auth.forgotPassword} />
      </AuthShell>
    )
  }

  return (
    <AuthShell
      title="Создайте новый пароль"
      subtitle="Придумайте новый пароль для своего аккаунта. После сохранения вы сможете войти в Физтех.Ответ."
    >
      <ResetPasswordForm token={token} loginHref={paths.auth.login} onDone={setResult} />
    </AuthShell>
  )
}

function ResetFail({ forgotHref }: { forgotHref: string }) {
  return (
    <div className="pl-auth__done">
      <Link to={forgotHref} className="pl-btn pl-btn--primary pl-auth__submit">
        Запросить новую ссылку
      </Link>
      <p className="pl-auth__alt">
        <Link to={paths.auth.login}>Вернуться ко входу</Link>
      </p>
    </div>
  )
}
