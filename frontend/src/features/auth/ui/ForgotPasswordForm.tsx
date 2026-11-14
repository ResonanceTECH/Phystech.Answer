import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { authApi } from '@/entities/user'
import { authErrorMessage } from '../lib/errors'
import { AuthField } from './AuthField'

const schema = z.object({
  login: z.string().trim().min(1, 'Введите логин или email'),
})
type FormValues = z.infer<typeof schema>

interface Props {
  loginHref: string
}

export function ForgotPasswordForm({ loginHref }: Props) {
  const [sentTo, setSentTo] = useState<string | null>(null)
  const { register, handleSubmit, setError, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { login: '' },
  })

  const onSubmit = handleSubmit(async (values) => {
    try {
      await authApi.forgotPassword(values.login.trim())
      setSentTo(values.login.trim())
    } catch (err) {
      setError('root', { message: authErrorMessage(err) })
    }
  })

  if (sentTo) {
    return (
      <div className="pl-auth__done">
        <p className="pl-auth__alert pl-auth__alert--ok" role="status">
          Если аккаунт с логином «{sentTo}» существует, инструкция по сбросу
          пароля уже отправлена.
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
        id="forgot-login"
        label="Логин или email"
        error={formState.errors.login?.message}
      >
        <input
          id="forgot-login"
          className="pl-field__input"
          type="text"
          autoComplete="username"
          placeholder="ivan"
          {...register('login')}
        />
      </AuthField>

      <button
        type="submit"
        className="pl-btn pl-btn--primary pl-auth__submit"
        disabled={formState.isSubmitting}
      >
        {formState.isSubmitting ? 'Отправляем…' : 'Отправить инструкцию'}
      </button>
    </form>
  )
}
