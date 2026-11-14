import { useEffect, useRef, useState } from 'react'
import { useLocation } from 'react-router-dom'
import { api } from '@/api/client'
import type { AskResponse, AskSource } from '@/api/types'

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  text: string
  sources?: AskSource[]
  stub?: boolean
  ts: number
}

const STORAGE_KEY = 'phystech.chat.v1'
const MAX_STORED = 100

const SUGGESTIONS = [
  'Мне нужно поехать на научную конференцию от МФТИ. Какие документы оформить?',
  'Как открыть финансово-лицевой счёт для научного проекта?',
  'Хочу нанять сотрудника в подразделение — с чего начать?',
  'Сломался рабочий компьютер — куда обращаться?',
  'Как оформить внутреннее заимствование средств?',
]

function loadHistory(): ChatMessage[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return []
    const parsed = JSON.parse(raw) as ChatMessage[]
    return Array.isArray(parsed) ? parsed.slice(-MAX_STORED) : []
  } catch {
    return []
  }
}

let nextId = 0
function msgId() {
  nextId += 1
  return `m${Date.now().toString(36)}-${nextId}`
}

export function ChatPage() {
  const [messages, setMessages] = useState<ChatMessage[]>(loadHistory)
  const [input, setInput] = useState('')
  const [pending, setPending] = useState(false)
  const location = useLocation()
  const bottomRef = useRef<HTMLDivElement>(null)
  const autoSentRef = useRef(false)

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(messages.slice(-MAX_STORED)))
    } catch {
      // storage full or unavailable — history is best-effort
    }
  }, [messages])

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, pending])

  // Auto-send a question passed via router state (e.g. from a document page).
  useEffect(() => {
    const state = location.state as { question?: string } | null
    if (state?.question && !autoSentRef.current) {
      autoSentRef.current = true
      void send(state.question)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [location.state])

  async function send(text: string) {
    const question = text.trim()
    if (!question || pending) return

    const userMsg: ChatMessage = { id: msgId(), role: 'user', text: question, ts: Date.now() }
    setMessages((prev) => [...prev, userMsg])
    setInput('')
    setPending(true)

    try {
      const resp = await api.post<AskResponse>('/ask', { question })
      setMessages((prev) => [
        ...prev,
        {
          id: msgId(),
          role: 'assistant',
          text: resp.answer,
          sources: resp.sources ?? [],
          stub: resp.stub,
          ts: Date.now(),
        },
      ])
    } catch (err) {
      setMessages((prev) => [
        ...prev,
        {
          id: msgId(),
          role: 'assistant',
          text: `Не удалось получить ответ: ${err instanceof Error ? err.message : 'ошибка сервера'}`,
          ts: Date.now(),
        },
      ])
    } finally {
      setPending(false)
    }
  }

  function onSubmit(e: React.FormEvent) {
    e.preventDefault()
    void send(input)
  }

  function clearChat() {
    setMessages([])
    localStorage.removeItem(STORAGE_KEY)
  }

  return (
    <div className="flex h-[calc(100svh-7.5rem)] flex-col gap-4">
      <div className="flex items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Ассистент</h1>
          <p className="mt-1 text-sm text-neutral-600">
            Задайте вопрос о внутренних регламентах и административных
            процедурах МФТИ — ответ будет со ссылками на документы.
          </p>
        </div>
        {messages.length > 0 && (
          <button
            type="button"
            onClick={clearChat}
            className="shrink-0 rounded-md border border-neutral-300 px-3 py-1.5 text-sm text-neutral-600 hover:bg-neutral-100"
          >
            Очистить
          </button>
        )}
      </div>

      <div className="flex min-h-0 flex-1 flex-col rounded-xl border border-neutral-200 bg-white">
        <div className="flex-1 overflow-y-auto px-4 py-4">
          {messages.length === 0 && !pending ? (
            <div className="flex h-full flex-col items-center justify-center gap-4 text-center">
              <p className="max-w-md text-sm text-neutral-500">
                Например:
              </p>
              <div className="flex max-w-lg flex-wrap justify-center gap-2">
                {SUGGESTIONS.map((s) => (
                  <button
                    key={s}
                    type="button"
                    onClick={() => void send(s)}
                    className="rounded-full border border-neutral-300 bg-neutral-50 px-3 py-1.5 text-left text-sm text-neutral-700 hover:border-neutral-400 hover:bg-neutral-100"
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
          ) : (
            <div className="flex flex-col gap-4">
              {messages.map((m) => (
                <MessageBubble key={m.id} message={m} />
              ))}
              {pending && (
                <div className="flex items-center gap-2 text-sm text-neutral-500">
                  <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-neutral-400" />
                  Ассистент формирует ответ…
                </div>
              )}
              <div ref={bottomRef} />
            </div>
          )}
        </div>

        <form
          onSubmit={onSubmit}
          className="flex items-end gap-2 border-t border-neutral-200 p-3"
        >
          <textarea
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && !e.shiftKey) {
                e.preventDefault()
                void send(input)
              }
            }}
            rows={2}
            placeholder="Опишите вашу задачу обычными словами…"
            className="flex-1 resize-none rounded-lg border border-neutral-300 px-3 py-2 text-sm outline-none focus:border-neutral-500"
          />
          <button
            type="submit"
            disabled={pending || !input.trim()}
            className="rounded-lg bg-neutral-900 px-4 py-2 text-sm font-medium text-white disabled:opacity-40"
          >
            Спросить
          </button>
        </form>
      </div>
    </div>
  )
}

function MessageBubble({ message }: { message: ChatMessage }) {
  if (message.role === 'user') {
    return (
      <div className="flex justify-end">
        <div className="max-w-[80%] whitespace-pre-wrap rounded-2xl rounded-br-md bg-neutral-900 px-4 py-2.5 text-sm text-white">
          {message.text}
        </div>
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-2">
      <div className="max-w-[90%] rounded-2xl rounded-bl-md border border-neutral-200 bg-neutral-50 px-4 py-3 text-sm text-neutral-900">
        <div className="whitespace-pre-wrap">{message.text}</div>
        {message.stub && (
          <div className="mt-2 inline-block rounded-full bg-amber-100 px-2 py-0.5 text-xs text-amber-800">
            демо-ответ: ML-сервис недоступен
          </div>
        )}
      </div>
      {message.sources && message.sources.length > 0 && (
        <div className="flex max-w-[90%] flex-col gap-2">
          <span className="text-xs font-medium uppercase tracking-wide text-neutral-400">
            Источники
          </span>
          {message.sources.map((src, i) => (
            <SourceCard key={`${src.documentId}-${i}`} source={src} index={i + 1} />
          ))}
        </div>
      )}
    </div>
  )
}

function SourceCard({ source, index }: { source: AskSource; index: number }) {
  return (
    <a
      href={`/documents/${source.documentId}`}
      className="block rounded-lg border border-neutral-200 bg-white p-3 text-sm hover:border-neutral-400"
    >
      <div className="flex items-baseline justify-between gap-2">
        <span className="font-medium text-neutral-900">
          [{index}] {source.title}
        </span>
        {source.section && (
          <span className="shrink-0 rounded bg-neutral-100 px-1.5 py-0.5 text-xs text-neutral-600">
            п. {source.section}
          </span>
        )}
      </div>
      <p className="mt-1 line-clamp-3 text-xs text-neutral-600">{source.excerpt}</p>
    </a>
  )
}
