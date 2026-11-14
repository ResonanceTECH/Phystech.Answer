import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import { useAuthStore, type User } from '@/entities/user'
import { authErrorMessage } from '../lib/errors'
import { AuthField } from './AuthField'
import { PasswordInput } from './PasswordInput'
import { PasswordStrength } from './PasswordStrength'

const schema = z
  .object({
    firstName: z.string().trim().min(1, 'Введите имя').max(80, 'Слишком длинное имя'),
    lastName: z.string().trim().min(1, 'Введите фамилию').max(80, 'Слишком длинная фамилия'),
    email: z.email('Некорректный email'),
    password: z.string().min(12, 'Пароль от 12 символов'),
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
  const registerUser = useAuthStore((s) => s.register)
  const { register, handleSubmit, setError, watch, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { firstName: '', lastName: '', email: '', password: '', confirm: '' },
  })
  const password = watch('password')

  const onSubmit = handleSubmit(async (values) => {
    try {
      const user = await registerUser({
        firstName: values.firstName.trim(),
        lastName: values.lastName.trim(),
        email: values.email.trim(),
        password: values.password,
      })
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

      <AuthField
        id="reg-first-name"
        label="Имя"
        error={formState.errors.firstName?.message}
      >
        <input
          id="reg-first-name"
          className="pl-field__input"
          type="text"
          autoComplete="given-name"
          placeholder="Александр"
          {...register('firstName')}
        />
      </AuthField>

      <AuthField
        id="reg-last-name"
        label="Фамилия"
        error={formState.errors.lastName?.message}
      >
        <input
          id="reg-last-name"
          className="pl-field__input"
          type="text"
          autoComplete="family-name"
          placeholder="Иванов"
          {...register('lastName')}
        />
      </AuthField>

      <AuthField id="reg-email" label="Электронная почта" error={formState.errors.email?.message}>
        <input
          id="reg-email"
          className="pl-field__input"
          type="email"
          autoComplete="email"
          placeholder="name@mipt.ru"
          {...register('email')}
        />
      </AuthField>

      <AuthField
        id="reg-password"
        label="Пароль"
        error={formState.errors.password?.message}
      >
        <PasswordInput
          id="reg-password"
          autoComplete="new-password"
          placeholder="От 12 символов"
          error={!!formState.errors.password}
          {...register('password')}
        />
        <PasswordStrength password={password} />
      </AuthField>

      <AuthField
        id="reg-confirm"
        label="Повторите пароль"
        error={formState.errors.confirm?.message}
      >
        <PasswordInput
          id="reg-confirm"
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
        {formState.isSubmitting ? 'Создаём…' : 'Создать аккаунт'}
      </button>
    </form>
  )
}
