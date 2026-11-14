import { Link } from 'react-router-dom'

export function NotFoundPage() {
  return (
    <div className="flex flex-col items-start gap-3">
      <h1 className="text-2xl font-semibold tracking-tight">
        Страница не найдена
      </h1>
      <Link to="/" className="text-sm text-neutral-600 underline">
        На главную
      </Link>
    </div>
  )
}
