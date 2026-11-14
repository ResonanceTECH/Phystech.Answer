import { useQuery } from '@tanstack/react-query'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { api } from '@/api/client'
import type { Document } from '@/api/types'

export function DocumentPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const document = useQuery({
    queryKey: ['document', id],
    queryFn: () => api.get<Document>(`/documents/${id}`),
    enabled: !!id,
  })

  const doc = document.data

  function askAssistant() {
    if (!doc) return
    navigate('/', {
      state: {
        question: `Расскажите про «${doc.title}»: порядок действий, необходимые документы, сроки и ответственные. Что нужно знать сотруднику?`,
      },
    })
  }

  return (
    <div className="flex flex-col gap-4">
      <Link to="/documents" className="text-sm text-neutral-500 hover:text-neutral-800">
        ← Все документы
      </Link>

      {document.isPending && <p className="text-sm text-neutral-500">Загрузка…</p>}
      {document.isError && (
        <p className="text-sm text-red-600">Документ не найден или недоступен.</p>
      )}

      {doc && (
        <>
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h1 className="text-2xl font-semibold tracking-tight">{doc.title}</h1>
              <div className="mt-2 flex flex-wrap items-center gap-2 text-xs text-neutral-500">
                <span>ред. {doc.version}</span>
                <span>·</span>
                <span>
                  обновлён {new Date(doc.updatedAt).toLocaleString('ru-RU')}
                </span>
                {doc.tags.map((t) => (
                  <span
                    key={t}
                    className="rounded-full bg-neutral-100 px-2 py-0.5 text-neutral-600"
                  >
                    {t}
                  </span>
                ))}
              </div>
            </div>
            <button
              type="button"
              onClick={askAssistant}
              className="rounded-lg bg-neutral-900 px-4 py-2 text-sm font-medium text-white"
            >
              Обсудить с ассистентом
            </button>
          </div>

          <div className="rounded-xl border border-neutral-200 bg-white p-5">
            <div className="whitespace-pre-wrap text-sm leading-relaxed text-neutral-800">
              {doc.content}
            </div>
          </div>
        </>
      )}
    </div>
  )
}
