import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '@/shared/api'
import type { Assignment, AssignRequirement, Department, Participant, TargetType, Visibility } from '../model/types'

const keys = {
  departments: ['admin', 'departments'] as const,
  participants: ['admin', 'participants'] as const,
  assignments: (documentId = '', targetType = '', targetId = '') =>
    ['admin', 'assignments', documentId, targetType, targetId] as const,
}

function useInvalidate() {
  const qc = useQueryClient()
  return (...prefixes: readonly unknown[][]) =>
    Promise.all(prefixes.map((queryKey) => qc.invalidateQueries({ queryKey })))
}

// ---------- documents ----------

export interface DocumentPatch {
  title?: string
  folder?: string
  visibility?: Visibility
}

export function usePatchDocuments() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: ({ ids, patch }: { ids: string[]; patch: DocumentPatch }) =>
      Promise.all(ids.map((id) => api.patch(`/v1/admin/documents/${id}`, patch))),
    onSuccess: () => invalidate(['documents'], ['reading-plan']),
  })
}

export function useDeleteDocuments() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: (ids: string[]) => Promise.all(ids.map((id) => api.delete(`/v1/documents/${id}`))),
    onSuccess: () => invalidate(['documents'], ['reading-plan'], ['admin']),
  })
}

export function useReprocessDocument() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: (id: string) => api.post<{ jobId: string; status: string }>(`/v1/admin/documents/${id}/reprocess`, {}),
    onSuccess: () => invalidate(['documents'], ['admin', 'processing']),
  })
}

// ---------- departments ----------

export function useDepartments() {
  return useQuery({ queryKey: keys.departments, queryFn: () => api.get<Department[]>('/v1/departments') })
}

export function useCreateDepartment() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: (name: string) => api.post<Department>('/v1/admin/departments', { name }),
    onSuccess: () => invalidate([...keys.departments]),
  })
}

export function useRenameDepartment() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: ({ id, name }: { id: string; name: string }) => api.patch(`/v1/admin/departments/${id}`, { name }),
    onSuccess: () => invalidate([...keys.departments], [...keys.participants]),
  })
}

export function useDeleteDepartment() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/admin/departments/${id}`),
    onSuccess: () => invalidate(['admin']),
  })
}

// ---------- participants ----------

export function useParticipants() {
  return useQuery({ queryKey: keys.participants, queryFn: () => api.get<Participant[]>('/v1/admin/users') })
}

export interface ParticipantPatch {
  role?: Participant['role']
  departmentId?: string
  position?: string
  active?: boolean
}

export function usePatchParticipant() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: ({ id, patch }: { id: string; patch: ParticipantPatch }) =>
      api.patch<Participant>(`/v1/admin/users/${id}`, patch),
    onSuccess: () => invalidate([...keys.participants], [...keys.departments]),
  })
}

export interface NewParticipant {
  name: string
  login: string
  email?: string
  password: string
  role?: Participant['role']
  departmentId?: string
  position?: string
}

export function useCreateParticipant() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: (p: NewParticipant) => api.post<Participant>('/v1/admin/users', p),
    onSuccess: () => invalidate([...keys.participants], [...keys.departments]),
  })
}

// ---------- assignments ----------

export function useAssignments(filter: { documentId?: string; targetType?: TargetType; targetId?: string } = {}) {
  const qs = new URLSearchParams()
  if (filter.documentId) qs.set('documentId', filter.documentId)
  if (filter.targetType) qs.set('targetType', filter.targetType)
  if (filter.targetId) qs.set('targetId', filter.targetId)
  return useQuery({
    queryKey: keys.assignments(filter.documentId, filter.targetType, filter.targetId),
    queryFn: () => api.get<Assignment[]>(`/v1/admin/assignments?${qs.toString()}`),
  })
}

export interface AssignInput {
  documentIds: string[]
  targetType: TargetType
  targetId?: string
  requirement: AssignRequirement
  dueDate?: string
}

export function useAssign() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: (input: AssignInput) => api.post<Assignment[]>('/v1/admin/assignments', input),
    onSuccess: () => invalidate(['admin'], ['reading-plan']),
  })
}

export function useDeleteAssignment() {
  const invalidate = useInvalidate()
  return useMutation({
    mutationFn: (id: string) => api.delete(`/v1/admin/assignments/${id}`),
    onSuccess: () => invalidate(['admin'], ['reading-plan']),
  })
}
