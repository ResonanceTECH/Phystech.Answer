import { useQuery } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { api } from '@/api/client'
import type { Document } from '@/api/types'

export function DocumentsPage() {
  const [search, setSearch] = useState('')

  const documents = useQuery({
    queryKey: ['documents'],
    queryFn: () => api.get<Document[]>('/documents'),
  })

  const filtered = useMemo(() => {
    const docs = documents.data ?? []
    const q = search.trim().toLowerCase()
    if (!q) return docs
    return docs.filter(
      (d) =>
        d.title.toLowerCase().includes(q) ||
        d.content.toLowerCase().includes(q) ||
        d.tags.some((t) => t.toLowerCase().includes(q)),
    )
  }, [documents.data, search])

  return (
    <div className="flex flex-col gap-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">Документы</h1>
        <p className="mt-1 text-sm text-neutral-600">
          Нормативная база: регламенты, приказы и инструкции МФТИ.
        </p>
      </div>

      <input
        type="search"
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        placeholder="Поиск по названию, тегам и содержимому…"
        className="w-full rounded-lg border border-neutral-300 bg-white px-3 py-2 text-sm outline-none focus:border-neutral-500"
      />

      {documents.isPending && (
        <p className="text-sm text-neutral-500">Загрузка…</p>
      )}
      {documents.isError && (
        <p className="text-sm text-red-600">Не удалось загрузить документы.</p>
      )}
      {documents.isSuccess && filtered.length === 0 && (
        <p className="text-sm text-neutral-500">
          {search ? 'Ничего не найдено.' : 'База документов пуста.'}
        </p>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        {filtered.map((doc) => (
          <Link
            key={doc.id}
            to={`/documents/${doc.id}`}
            className="flex flex-col gap-2 rounded-xl border border-neutral-200 bg-white p-4 hover:border-neutral-400"
          >
            <div className="flex items-start justify-between gap-2">
              <h2 className="font-medium leading-snug text-neutral-900">
                {doc.title}
              </h2>
              {doc.read && (
                <span className="shrink-0 rounded-full bg-emerald-50 px-2 py-0.5 text-xs text-emerald-700">
                  прочитан
                </span>
              )}
            </div>
            <div className="flex flex-wrap gap-1.5">
              {doc.tags.map((t) => (
                <span
                  key={t}
                  className="rounded-full bg-neutral-100 px-2 py-0.5 text-xs text-neutral-600"
                >
                  {t}
                </span>
              ))}
            </div>
            <span className="text-xs text-neutral-400">
              ред. {doc.version} · обновлён{' '}
              {new Date(doc.updatedAt).toLocaleDateString('ru-RU')}
            </span>
          </Link>
        ))}
      </div>
    </div>
  )
}
