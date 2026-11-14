// Централизованные пути маршрутов — единственное место, где
// собираются URL приложения.
export const paths = {
  home: '/',
  auth: {
    login: '/login',
    register: '/register',
    forgotPassword: '/forgot-password',
    resetPassword: '/reset-password',
    onboarding: '/onboarding',
  },
  app: {
    root: '/app',
    home: '/app',
    profile: '/app/profile',
    chatNew: '/app/chat/new',
    chat: (id: string) => `/app/chat/${id}`,
    documents: '/app/documents',
    document: (id: string) => `/app/documents/${id}`,
    instructions: '/app/instructions',
    plan: '/app/plan',
  },
  admin: {
    root: '/admin',
    dashboard: '/admin',
    documents: '/admin/documents',
    upload: '/admin/documents/upload',
    participants: '/admin/participants',
    processing: '/admin/processing',
    versions: '/admin/versions',
    conflicts: '/admin/conflicts',
    analytics: '/admin/analytics',
  },
} as const
