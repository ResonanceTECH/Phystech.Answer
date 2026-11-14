// Централизованные ключи TanStack Query. Группировка по сущностям
// позволяет точечно инвалидировать кэш после мутаций.
export const queryKeys = {
  health: {
    all: ['health'] as const,
  },
  documents: {
    all: ['documents'] as const,
    list: (userId?: string) => ['documents', 'list', userId ?? ''] as const,
    detail: (id: string) => ['documents', 'detail', id] as const,
  },
  users: {
    all: ['users'] as const,
    list: () => ['users', 'list'] as const,
    unread: (id: string) => ['users', 'unread', id] as const,
  },
} as const
