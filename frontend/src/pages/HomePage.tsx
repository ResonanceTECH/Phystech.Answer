import { useQuery } from '@tanstack/react-query'
import { api } from '@/api/client'

type Health = { status: string }

export function HomePage() {
  const health = useQuery({
    queryKey: ['health'],
    queryFn: () => api.get<Health>('/health'),
  })

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">
          Навигатор по непрочитанным регламентам
        </h1>
        <p className="mt-1 text-sm text-neutral-600">
          База фронтенда: React, TypeScript, Tailwind, React Router, TanStack
          Query.
        </p>
      </div>

      <div className="rounded-lg border border-neutral-200 bg-white p-4 text-sm">
        <span className="text-neutral-600">Статус бэкенда: </span>
        {health.isPending && <span>проверяем…</span>}
        {health.isError && (
          <span className="text-red-600">недоступен</span>
        )}
        {health.isSuccess && (
          <span className="text-emerald-600">{health.data.status}</span>
        )}
      </div>
    </div>
  )
}
