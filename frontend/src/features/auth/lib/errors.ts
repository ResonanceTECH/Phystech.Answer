import { apiErrorMessage, isApiError } from '@/shared/api/errors'

// Понятные тексты для форм авторизации: код API → конкретная причина,
// сетевая ошибка → «сервер недоступен».
export function authErrorMessage(err: unknown): string {
  if (isApiError(err)) {
    switch (err.code) {
      case 'INVALID_CREDENTIALS':
        return 'Не удалось войти. Проверьте почту и пароль.'
      case 'EMAIL_ALREADY_EXISTS':
        return 'Аккаунт с такой почтой уже существует.'
      case 'RATE_LIMITED':
        return 'Слишком много попыток. Подождите минуту и попробуйте снова.'
      case 'RESET_TOKEN_INVALID':
        return 'Не удалось подтвердить ссылку для восстановления.'
      case 'RESET_TOKEN_EXPIRED':
        return 'Срок действия ссылки истёк. Запросите новую.'
      case 'RESET_TOKEN_USED':
        return 'Эта ссылка уже использована.'
      case 'UNAUTHORIZED':
        return 'Сессия истекла. Войдите снова.'
      default:
        return apiErrorMessage(err)
    }
  }
  return 'Не удалось связаться с сервером. Попробуйте ещё раз.'
}
