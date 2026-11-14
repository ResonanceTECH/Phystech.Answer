import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/shared/api'

export interface AppNotification {
  id: string
  kind: 'assigned' | 'new_document' | 'updated'
  documentId?: string
  title: string
  body?: string
  readAt?: string
  createdAt: string
}

const key = ['notifications'] as const

export function useNotifications(enabled: boolean) {
  return useQuery({
    queryKey: key,
    queryFn: () => api.get<{ items: AppNotification[]; unread: number }>('/v1/notifications'),
    enabled,
    refetchInterval: 60_000,
    refetchOnWindowFocus: true,
  })
}

/** id — одно уведомление; без id — все. */
export function useMarkNotificationsRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id?: string) => api.post<void>('/v1/notifications/read', id ? { id } : {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: key }),
  })
}
