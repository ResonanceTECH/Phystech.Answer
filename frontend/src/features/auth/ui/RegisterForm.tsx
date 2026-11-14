import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { authApi, useAuthStore, type User } from '@/entities/user'
import { authErrorMessage } from '../lib/errors'
import { AuthField } from './AuthField'

const schema = z
  .object({
    name: z.string().trim().min(2, 'Введите имя'),
    login: z
      .string()
      .trim()
      .min(3, 'Логин от 3 символов')
      .regex(/^[a-zA-Z0-9_.-]+$/, 'Только латиница, цифры и _.-'),
    email: z.union([z.literal(''), z.email('Некорректный email')]),
    password: z.string().min(6, 'Пароль от 6 символов'),
    confirm: z.string(),
  })
  .refine((v) => v.password === v.confirm, {
    message: 'Пароли не совпадают',
    path: ['confirm'],
  })
type FormValues = z.infer<typeof schema>

interface Props {
  onSuccess?: (user: User) => void
}

export function RegisterForm({ onSuccess }: Props) {
  const setUser = useAuthStore((s) => s.setUser)
  const { register, handleSubmit, setError, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { name: '', login: '', email: '', password: '', confirm: '' },
  })

  const onSubmit = handleSubmit(async (values) => {
    try {
      const user = await authApi.register({
        name: values.name.trim(),
        login: values.login.trim(),
        email: values.email.trim() || undefined,
        password: values.password,
      })
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

      <AuthField id="reg-name" label="Имя" error={formState.errors.name?.message}>
        <input
          id="reg-name"
          className="pl-field__input"
          type="text"
          autoComplete="name"
          placeholder="Иван Петров"
          {...register('name')}
        />
      </AuthField>

      <AuthField id="reg-login" label="Логин" error={formState.errors.login?.message}>
        <input
          id="reg-login"
          className="pl-field__input"
          type="text"
          autoComplete="username"
          placeholder="ivan.petrov"
          {...register('login')}
        />
      </AuthField>

      <AuthField
        id="reg-email"
        label="Email (необязательно)"
        error={formState.errors.email?.message}
      >
        <input
          id="reg-email"
          className="pl-field__input"
          type="email"
          autoComplete="email"
          placeholder="ivan@phystech.edu"
          {...register('email')}
        />
      </AuthField>

      <AuthField
        id="reg-password"
        label="Пароль"
        error={formState.errors.password?.message}
      >
        <input
          id="reg-password"
          className="pl-field__input"
          type="password"
          autoComplete="new-password"
          placeholder="От 6 символов"
          {...register('password')}
        />
      </AuthField>

      <AuthField
        id="reg-confirm"
        label="Повторите пароль"
        error={formState.errors.confirm?.message}
      >
        <input
          id="reg-confirm"
          className="pl-field__input"
          type="password"
          autoComplete="new-password"
          placeholder="Ещё раз"
          {...register('confirm')}
        />
      </AuthField>

      <button
        type="submit"
        className="pl-btn pl-btn--primary pl-auth__submit"
        disabled={formState.isSubmitting}
      >
        {formState.isSubmitting ? 'Создаём…' : 'Зарегистрироваться'}
      </button>
    </form>
  )
}
