import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { authApi } from '@/entities/user'
import { isApiError } from '@/shared/api/errors'
import { authErrorMessage } from '../lib/errors'
import { AuthField } from './AuthField'
import { PasswordInput } from './PasswordInput'
import { PasswordStrength } from './PasswordStrength'

const schema = z
  .object({
    password: z.string().min(12, 'Пароль от 12 символов'),
    confirm: z.string(),
  })
  .refine((v) => v.password === v.confirm, {
    message: 'Пароли не совпадают',
    path: ['confirm'],
  })
type FormValues = z.infer<typeof schema>

export type ResetResult =
  | { kind: 'success' }
  | { kind: 'invalid' | 'expired' | 'used'; message: string }

interface Props {
  token: string
  loginHref: string
  onDone: (result: ResetResult) => void
}

// Форма установки нового пароля по одноразовому токену.
export function ResetPasswordForm({ token, loginHref, onDone }: Props) {
  const [apiError, setApiError] = useState<string | null>(null)
  const { register, handleSubmit, setError, watch, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { password: '', confirm: '' },
  })
  const password = watch('password')

  const onSubmit = handleSubmit(async (values) => {
    setApiError(null)
    try {
      await authApi.resetPassword(token, values.password)
      onDone({ kind: 'success' })
    } catch (err) {
      if (isApiError(err) && err.code?.startsWith('RESET_TOKEN_')) {
        onDone({ kind: err.code === 'RESET_TOKEN_EXPIRED' ? 'expired' : err.code === 'RESET_TOKEN_USED' ? 'used' : 'invalid', message: authErrorMessage(err) })
        return
      }
      setApiError(authErrorMessage(err))
      setError('root', { message: authErrorMessage(err) })
    }
  })

  return (
    <form className="pl-auth__form" onSubmit={onSubmit} noValidate>
      {(apiError ?? formState.errors.root?.message) ? (
        <p className="pl-auth__alert" role="alert">
          {apiError ?? formState.errors.root?.message}
        </p>
      ) : null}

      <AuthField
        id="new-password"
        label="Новый пароль"
        error={formState.errors.password?.message}
      >
        <PasswordInput
          id="new-password"
          autoComplete="new-password"
          placeholder="От 12 символов"
          error={!!formState.errors.password}
          {...register('password')}
        />
        <PasswordStrength password={password} />
      </AuthField>

      <AuthField
        id="new-confirm"
        label="Повторите новый пароль"
        error={formState.errors.confirm?.message}
      >
        <PasswordInput
          id="new-confirm"
          autoComplete="new-password"
          placeholder="Ещё раз"
          error={!!formState.errors.confirm}
          {...register('confirm')}
        />
      </AuthField>

      <button
        type="submit"
        className="pl-btn pl-btn--primary pl-auth__submit"
        disabled={formState.isSubmitting}
      >
        {formState.isSubmitting ? 'Сохраняем…' : 'Сохранить новый пароль'}
      </button>

      <p className="pl-auth__alt">
        <Link to={loginHref}>Вернуться ко входу</Link>
      </p>
    </form>
  )
}
