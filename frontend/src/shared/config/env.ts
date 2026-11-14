// Единая конфигурация приложения из VITE_* переменных.
// В VITE_* нельзя хранить секреты — они попадают в клиентскую сборку.

const env = import.meta.env

export const config = {
  apiBaseUrl: env.VITE_API_URL ?? '/api',
  appName: env.VITE_APP_NAME ?? 'Физтех.Ответ',
  // Демо-режим: авторизация не подключена, админ-раздел доступен всем.
  // Должен быть явно обозначен в интерфейсе.
  demoMode: env.VITE_DEMO_MODE !== 'false',
  enableAdmin: env.VITE_ENABLE_ADMIN !== 'false',
} as const
