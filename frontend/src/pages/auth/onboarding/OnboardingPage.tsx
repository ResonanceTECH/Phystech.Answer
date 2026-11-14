import { useQuery } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate } from 'react-router-dom'
import { z } from 'zod'
import { homeFor } from '@/app/router/guards'
import { authApi, useAuthStore, type EmployeeCategory } from '@/entities/user'
import { authErrorMessage } from '@/features/auth/lib/errors'
import { AuthShell } from '../AuthShell'

const CATEGORIES: { value: EmployeeCategory; label: string }[] = [
  { value: 'employee', label: 'Сотрудник' },
  { value: 'teacher', label: 'Преподаватель' },
  { value: 'researcher', label: 'Исследователь' },
  { value: 'manager', label: 'Руководитель' },
]

const schema = z.object({
  employeeCategory: z.string().min(1, 'Выберите категорию'),
  departmentId: z.string(),
  position: z.string().trim().max(120, 'Слишком длинная должность'),
})
type FormValues = z.infer<typeof schema>

// Первичная настройка профиля после регистрации.
// Категория — для персонализации ответов, системных прав она не даёт.
export function OnboardingPage() {
  const navigate = useNavigate()
  const updateProfile = useAuthStore((s) => s.updateProfile)
  const user = useAuthStore((s) => s.user)

  const departments = useQuery({
    queryKey: ['departments'],
    queryFn: authApi.departments,
    staleTime: 5 * 60_000,
  })

  const { register, handleSubmit, setError, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { employeeCategory: '', departmentId: '', position: '' },
  })

  const onSubmit = handleSubmit(async (values) => {
    try {
      const updated = await updateProfile({
        employeeCategory: values.employeeCategory as EmployeeCategory,
        departmentId: values.departmentId || undefined,
        position: values.position.trim() || undefined,
        onboardingComplete: true,
      })
      navigate(homeFor(updated), { replace: true })
    } catch (err) {
      setError('root', { message: authErrorMessage(err) })
    }
  })

  return (
    <AuthShell
      title={`Расскажите о себе${user?.firstName ? `, ${user.firstName}` : ''}`}
      subtitle="Пара фактов о работе — ассистент будет точнее подбирать инструкции под вашу роль."
    >
      <form className="pl-auth__form" onSubmit={onSubmit} noValidate>
        {formState.errors.root?.message ? (
          <p className="pl-auth__alert" role="alert">
            {formState.errors.root.message}
          </p>
        ) : null}

        <div className="pl-field">
          <label className="pl-field__label" htmlFor="ob-category">
            Категория сотрудника
          </label>
          <select
            id="ob-category"
            className="pl-field__input pl-field__select"
            {...register('employeeCategory')}
          >
            <option value="" disabled>
              Выберите категорию
            </option>
            {CATEGORIES.map((c) => (
              <option key={c.value} value={c.value}>
                {c.label}
              </option>
            ))}
          </select>
          {formState.errors.employeeCategory?.message ? (
            <p className="pl-field__error" role="alert">
              {formState.errors.employeeCategory.message}
            </p>
          ) : null}
        </div>

        <div className="pl-field">
          <label className="pl-field__label" htmlFor="ob-department">
            Подразделение
          </label>
          <select
            id="ob-department"
            className="pl-field__input pl-field__select"
            disabled={departments.isLoading}
            {...register('departmentId')}
          >
            <option value="">
              {departments.isLoading ? 'Загружаем список…' : 'Не выбрано'}
            </option>
            {(departments.data ?? []).map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </select>
        </div>

        <div className="pl-field">
          <label className="pl-field__label" htmlFor="ob-position">
            Должность
          </label>
          <input
            id="ob-position"
            className="pl-field__input"
            type="text"
            placeholder="Например: инженер, доцент, аспирант"
            {...register('position')}
          />
        </div>

        <button
          type="submit"
          className="pl-btn pl-btn--primary pl-auth__submit"
          disabled={formState.isSubmitting}
        >
          {formState.isSubmitting ? 'Сохраняем…' : 'Перейти в кабинет'}
        </button>
      </form>
    </AuthShell>
  )
}
