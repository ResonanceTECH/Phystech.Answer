export class ApiError extends Error {
  status: number
  code?: string
  body?: unknown

  constructor(status: number, message: string, body?: unknown) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.body = body
    if (body && typeof body === 'object' && 'error' in body) {
      const e = (body as { error?: { code?: string } }).error
      if (e?.code) this.code = e.code
    }
  }
}

export function isApiError(err: unknown): err is ApiError {
  return err instanceof ApiError
}

// userMessage возвращает понятное пользователю описание ошибки API.
export function apiErrorMessage(err: unknown): string {
  if (isApiError(err)) {
    switch (err.status) {
      case 400:
        return 'Некорректный запрос.'
      case 401:
        return 'Требуется авторизация.'
      case 403:
        return 'Недостаточно прав.'
      case 404:
        return 'Ресурс не найден.'
      case 409:
        return 'Конфликт состояния — обновите данные.'
      case 422:
        return 'Ошибка введённых данных.'
      case 429:
        return 'Слишком много запросов. Попробуйте позже.'
      case 503:
        return 'Сервис временно недоступен.'
      default:
        return err.status >= 500
          ? 'Внутренняя ошибка сервера.'
          : `Ошибка запроса (${err.status}).`
    }
  }
  if (err instanceof Error && err.message) {
    return err.message
  }
  return 'Неизвестная ошибка.'
}
