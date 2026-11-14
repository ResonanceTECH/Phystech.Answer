// Централизованные пути маршрутов — единственное место, где
// собираются URL приложения.
export const paths = {
  home: '/',
  login: '/login',
  app: {
    root: '/app',
    home: '/app',
    chatNew: '/app/chat/new',
    chat: (id: string) => `/app/chat/${id}`,
    documents: '/app/documents',
    document: (id: string) => `/app/documents/${id}`,
    instructions: '/app/instructions',
    onboarding: '/app/onboarding',
  },
  admin: {
    root: '/admin',
    dashboard: '/admin',
    documents: '/admin/documents',
    upload: '/admin/documents/upload',
    processing: '/admin/processing',
    versions: '/admin/versions',
    conflicts: '/admin/conflicts',
    analytics: '/admin/analytics',
  },
} as const
