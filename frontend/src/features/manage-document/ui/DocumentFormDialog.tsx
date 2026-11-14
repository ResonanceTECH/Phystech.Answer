import {
  Alert,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Stack,
  TextField,
} from '@mui/material'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  useCreateDocument,
  useUpdateDocument,
  type Document,
} from '@/entities/document'

const schema = z.object({
  title: z.string().trim().min(2, 'Название слишком короткое').max(300),
  tags: z.string(),
  content: z.string().trim().min(20, 'Текст документа слишком короткий'),
})
type FormValues = z.infer<typeof schema>

interface Props {
  open: boolean
  onClose: () => void
  // undefined — создание, иначе редактирование существующего
  document?: Document
}

// Форма создания/редактирования документа (admin).
export function DocumentFormDialog({ open, onClose, document }: Props) {
  const createMutation = useCreateDocument()
  const updateMutation = useUpdateDocument(document?.id ?? '')
  const mutation = document ? updateMutation : createMutation

  const { register, handleSubmit, reset, formState } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { title: '', tags: '', content: '' },
  })

  useEffect(() => {
    if (open) {
      reset({
        title: document?.title ?? '',
        tags: document?.tags.join(', ') ?? '',
        content: document?.content ?? '',
      })
    }
  }, [open, document, reset])

  const onSubmit = handleSubmit((values) => {
    const payload = {
      title: values.title.trim(),
      content: values.content.trim(),
      tags: values.tags
        .split(',')
        .map((t) => t.trim())
        .filter(Boolean),
    }
    const onSuccess = () => {
      reset()
      onClose()
    }
    if (document) updateMutation.mutate(payload, { onSuccess })
    else createMutation.mutate(payload, { onSuccess })
  })

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle>{document ? 'Редактировать документ' : 'Новый документ'}</DialogTitle>
      <DialogContent dividers>
        {mutation.isError && (
          <Alert severity="error" sx={{ mb: 2 }}>
            {(mutation.error as Error).message}
          </Alert>
        )}
        <Stack spacing={2} sx={{ mt: 1 }}>
          <TextField
            label="Название документа"
            {...register('title')}
            error={!!formState.errors.title}
            helperText={formState.errors.title?.message}
            fullWidth
            autoFocus
          />
          <TextField
            label="Теги (через запятую)"
            {...register('tags')}
            helperText="Например: командировки, бухгалтерия"
            fullWidth
          />
          <TextField
            label="Текст документа"
            {...register('content')}
            error={!!formState.errors.content}
            helperText={formState.errors.content?.message ?? 'Разделы нумеруйте (1., 1.1, 2.) — так ассистенту проще находить нужные пункты.'}
            multiline
            minRows={10}
            fullWidth
          />
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose} disabled={mutation.isPending}>
          Отмена
        </Button>
        <Button
          onClick={onSubmit}
          variant="contained"
          disabled={mutation.isPending}
        >
          {mutation.isPending ? 'Сохранение…' : document ? 'Сохранить' : 'Создать'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
