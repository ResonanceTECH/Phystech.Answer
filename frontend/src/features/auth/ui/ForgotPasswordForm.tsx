import { useEffect, useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { authApi } from '@/entities/user'
import { authErrorMessage } from '../lib/errors'
import { AuthField } from './AuthField'

const schema = z.object({
  email: z.email('Введите корректную почту'),
})
type FormValues = z.infer<typeof schema>

const RESEND_COOLDOWN = 60

interface Props {
  loginHref: string
}

export function ForgotPasswordForm({ loginHref }: Props) {
  const [sentTo, setSentTo] = useState<string | null>(null)
  const [cooldown, setCooldown] = useState(0)
  const { register, handleSubmit, setError, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '' },
  })

  useEffect(() => {
    if (cooldown <= 0) return
    const t = setTimeout(() => setCooldown((c) => c - 1), 1000)
    return () => clearTimeout(t)
  }, [cooldown])

  const send = async (email: string) => {
    try {
      await authApi.forgotPassword(email)
      setSentTo(email)
      setCooldown(RESEND_COOLDOWN)
    } catch (err) {
      setError('root', { message: authErrorMessage(err) })
    }
  }

  const onSubmit = handleSubmit((values) => send(values.email.trim()))

  if (sentTo) {
    return (
      <div className="pl-auth__done">
        <div className="pl-auth__icon" aria-hidden>
          <svg viewBox="0 0 24 24" width="26" height="26" fill="none">
            <path
              d="M3 7a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V7Z"
              stroke="currentColor"
              strokeWidth="1.6"
            />
            <path
              d="m3.5 7 8.5 6 8.5-6M16 15l2.2 2.2L22.5 13"
              stroke="currentColor"
              strokeWidth="1.6"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>
        <p className="pl-auth__alert pl-auth__alert--ok" role="status">
          Если аккаунт с адресом «{sentTo}» существует, мы отправили письмо
          со ссылкой для восстановления доступа. Проверьте папку «Спам».
        </p>
        <p className="pl-auth__resend">
          Не пришло письмо?{' '}
          <button
            type="button"
            disabled={cooldown > 0 || formState.isSubmitting}
            onClick={() => send(sentTo)}
          >
            {cooldown > 0 ? `Отправить снова (${cooldown} с)` : 'Отправить снова'}
          </button>
        </p>
        <Link to={loginHref} className="pl-btn pl-btn--ghost pl-auth__submit">
          Вернуться ко входу
        </Link>
      </div>
    )
  }

  return (
    <form className="pl-auth__form" onSubmit={onSubmit} noValidate>
      {formState.errors.root?.message ? (
        <p className="pl-auth__alert" role="alert">
          {formState.errors.root.message}
        </p>
      ) : null}

      <AuthField
        id="forgot-email"
        label="Электронная почта"
        error={formState.errors.email?.message}
      >
        <input
          id="forgot-email"
          className="pl-field__input"
          type="email"
          autoComplete="email"
          placeholder="name@mipt.ru"
          {...register('email')}
        />
      </AuthField>

      <button
        type="submit"
        className="pl-btn pl-btn--primary pl-auth__submit"
        disabled={formState.isSubmitting}
      >
        {formState.isSubmitting ? 'Отправляем…' : 'Отправить ссылку'}
      </button>
    </form>
  )
}
