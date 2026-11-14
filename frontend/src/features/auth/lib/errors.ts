import { apiErrorMessage, isApiError } from '@/shared/api/errors'

// Понятные тексты для форм авторизации: статус → конкретная причина,
// сетевая ошибка → «сервер недоступен».
export function authErrorMessage(err: unknown): string {
  if (isApiError(err)) {
    if (err.status === 401) return 'Неверный логин или пароль.'
    if (err.status === 409) return 'Такой логин уже занят.'
    return apiErrorMessage(err)
  }
  return 'Не удалось связаться с сервером. Попробуйте ещё раз.'
}
