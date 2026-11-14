import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { authApi, useAuthStore, type User } from '@/entities/user'
import { authErrorMessage } from '../lib/errors'
import { AuthField } from './AuthField'

const schema = z.object({
  login: z.string().trim().min(1, 'Введите логин'),
  password: z.string().min(1, 'Введите пароль'),
})
type FormValues = z.infer<typeof schema>

interface Props {
  forgotPasswordHref: string
  onSuccess?: (user: User) => void
}

export function LoginForm({ forgotPasswordHref, onSuccess }: Props) {
  const setUser = useAuthStore((s) => s.setUser)
  const { register, handleSubmit, setError, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { login: '', password: '' },
  })

  const onSubmit = handleSubmit(async (values) => {
    try {
      const user = await authApi.login(values)
      setUser(user)
      onSuccess?.(user)
    } catch (err) {
      setError('root', { message: authErrorMessage(err) })
    }
  })

  return (
    <form className="pl-auth__form" onSubmit={onSubmit} noValidate>
      {formState.errors.root?.message ? (
        <p className="pl-auth__alert" role="alert">
          {formState.errors.root.message}
        </p>
      ) : null}

      <AuthField id="login" label="Логин" error={formState.errors.login?.message}>
        <input
          id="login"
          className="pl-field__input"
          type="text"
          autoComplete="username"
          placeholder="ivan"
          {...register('login')}
        />
      </AuthField>

      <AuthField
        id="password"
        label="Пароль"
        error={formState.errors.password?.message}
        aside={
          <Link className="pl-field__link" to={forgotPasswordHref}>
            Забыли пароль?
          </Link>
        }
      >
        <input
          id="password"
          className="pl-field__input"
          type="password"
          autoComplete="current-password"
          placeholder="••••••••"
          {...register('password')}
        />
      </AuthField>

      <button
        type="submit"
        className="pl-btn pl-btn--primary pl-auth__submit"
        disabled={formState.isSubmitting}
      >
        {formState.isSubmitting ? 'Входим…' : 'Войти'}
      </button>
    </form>
  )
}
