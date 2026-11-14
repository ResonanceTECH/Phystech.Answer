import { useCallback, useState } from 'react'
import { savedInstructionApi } from '@/entities/conversation'
import { apiErrorMessage } from '@/shared/api'

// Действие «Сохранить инструкцию»: ответ ассистента (answer_id) →
// серверная запись /saved-instructions, видна на всех устройствах.
export function useSaveInstruction() {
  const [pending, setPending] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const save = useCallback(async (answerId: string, title: string) => {
    if (!answerId || pending) return false
    setPending(true)
    setError(null)
    try {
      await savedInstructionApi.create(answerId, title)
      return true
    } catch (err) {
      setError(apiErrorMessage(err))
      return false
    } finally {
      setPending(false)
    }
  }, [pending])

  return { save, pending, error }
}
