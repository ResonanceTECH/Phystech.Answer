import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { Link } from 'react-router-dom'
import { z } from 'zod'
import { useAuthStore, type User } from '@/entities/user'
import { authErrorMessage } from '../lib/errors'
import { AuthField } from './AuthField'
import { PasswordInput } from './PasswordInput'

const schema = z.object({
  email: z.string().trim().min(1, 'Введите почту'),
  password: z.string().min(1, 'Введите пароль'),
  remember: z.boolean(),
})
type FormValues = z.infer<typeof schema>

interface Props {
  forgotPasswordHref: string
  onSuccess?: (user: User) => void
}

export function LoginForm({ forgotPasswordHref, onSuccess }: Props) {
  const login = useAuthStore((s) => s.login)
  const { register, handleSubmit, setError, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: '', password: '', remember: false },
  })

  const onSubmit = handleSubmit(async (values) => {
    try {
      const user = await login(values)
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

      <AuthField id="email" label="Электронная почта" error={formState.errors.email?.message}>
        <input
          id="email"
          className="pl-field__input"
          type="email"
          autoComplete="username"
          placeholder="name@mipt.ru"
          {...register('email')}
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
        <PasswordInput
          id="password"
          autoComplete="current-password"
          placeholder="••••••••"
          error={!!formState.errors.password}
          {...register('password')}
        />
      </AuthField>

      <label className="pl-check">
        <input type="checkbox" {...register('remember')} />
        <span className="pl-check__box" aria-hidden>
          <svg viewBox="0 0 12 12" width="10" height="10" fill="none">
            <path
              d="M2 6.2 4.8 9 10 3"
              stroke="currentColor"
              strokeWidth="1.8"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </span>
        Запомнить меня
      </label>

      <button
        type="submit"
        className="pl-btn pl-btn--primary pl-auth__submit"
        disabled={formState.isSubmitting}
      >
        {formState.isSubmitting ? 'Входим…' : 'Войти в аккаунт'}
      </button>
    </form>
  )
}
