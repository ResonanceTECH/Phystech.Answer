import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/shared/api'
import type { PlanResponse } from '../model/types'

export const planKey = ['reading-plan'] as const

export function usePlan() {
  return useQuery({ queryKey: planKey, queryFn: () => api.get<PlanResponse>('/v1/me/plan') })
}

/** Галочка «Ознакомлен»: отметка с датой (POST /documents/{id}/read). */
export function useAcknowledge() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (documentId: string) => api.post<{ readAt: string }>(`/v1/documents/${documentId}/read`),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: planKey })
      void qc.invalidateQueries({ queryKey: ['documents'] })
    },
  })
}
