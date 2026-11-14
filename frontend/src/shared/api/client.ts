import { config } from '@/shared/config/env'
import { ApiError } from './errors'

const BASE_URL = config.apiBaseUrl

async function request<T>(
  path: string,
  init?: RequestInit,
): Promise<T> {
  const isForm = init?.body instanceof FormData
  const res = await fetch(`${BASE_URL}${path}`, {
    ...init,
    credentials: 'include',
    // Для FormData заголовок с boundary выставит браузер.
    headers: isForm ? init?.headers : { 'Content-Type': 'application/json', ...init?.headers },
  })

  if (!res.ok) {
    let body: unknown
    let message = res.statusText
    try {
      body = await res.json()
      const parsed = body as {
        error?: string | { code?: string; message?: string }
        message?: string
      }
      message =
        (typeof parsed.error === 'object' ? parsed.error?.message : parsed.error) ??
        parsed.message ??
        message
    } catch {
      message = await res.text().catch(() => res.statusText)
    }
    throw new ApiError(res.status, message, body)
  }

  if (res.status === 204) {
    return undefined as T
  }

  return res.json() as Promise<T>
}

export const api = {
  get: <T>(path: string) => request<T>(path),
  post: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'POST', body: JSON.stringify(body) }),
  put: <T>(path: string, body?: unknown) =>
    request<T>(path, { method: 'PUT', body: JSON.stringify(body) }),
  delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
  postForm: <T>(path: string, form: FormData) => request<T>(path, { method: 'POST', body: form }),
}
