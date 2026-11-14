import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { queryKeys } from '@/shared/api'
import {
  createDocument,
  deleteDocument,
  getDocument,
  listDocuments,
  updateDocument,
} from '../api/documentApi'
import type { DocumentInput } from './types'

export function useDocuments(userId?: string) {
  return useQuery({
    queryKey: queryKeys.documents.list(userId),
    queryFn: () => listDocuments(userId),
  })
}

export function useDocument(id: string | undefined) {
  return useQuery({
    queryKey: queryKeys.documents.detail(id ?? ''),
    queryFn: () => getDocument(id as string),
    enabled: !!id,
  })
}

function useInvalidateDocuments() {
  const qc = useQueryClient()
  return () => qc.invalidateQueries({ queryKey: queryKeys.documents.all })
}

export function useCreateDocument() {
  const invalidate = useInvalidateDocuments()
  return useMutation({
    mutationFn: (input: DocumentInput) => createDocument(input),
    onSuccess: invalidate,
  })
}

export function useUpdateDocument(id: string) {
  const invalidate = useInvalidateDocuments()
  return useMutation({
    mutationFn: (input: DocumentInput) => updateDocument(id, input),
    onSuccess: invalidate,
  })
}

export function useDeleteDocument() {
  const invalidate = useInvalidateDocuments()
  return useMutation({
    mutationFn: (id: string) => deleteDocument(id),
    onSuccess: invalidate,
  })
}
