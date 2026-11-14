import CloudUploadIcon from '@mui/icons-material/CloudUpload'
import {
  Alert,
  Box,
  Button,
  Chip,
  LinearProgress,
  Paper,
  Typography,
} from '@mui/material'
import { useRef, useState } from 'react'
import {
  isSupportedTextFile,
  readTextFile,
  titleFromFileName,
} from '@/features/upload-documents'
import { useCreateDocument } from '@/entities/document'
import { PageHeader } from '@/shared/ui'

type FileStatus = 'pending' | 'processing' | 'done' | 'error' | 'unsupported'

interface Row {
  name: string
  status: FileStatus
  message?: string
}

// Загрузка документов из текстовых файлов (.txt/.md). PDF/DOCX —
// заглушка: backend пока не умеет их парсить, показываем честный статус.
export function DocumentUploadPage() {
  const inputRef = useRef<HTMLInputElement>(null)
  const createMutation = useCreateDocument()
  const [rows, setRows] = useState<Row[]>([])
  const [dragOver, setDragOver] = useState(false)

  const updateRow = (name: string, patch: Partial<Row>) =>
    setRows((prev) => prev.map((r) => (r.name === name ? { ...r, ...patch } : r)))

  const processFiles = async (files: File[]) => {
    const initial: Row[] = files.map((f) => ({ name: f.name, status: 'pending' }))
    setRows((prev) => [...prev, ...initial])

    for (const file of files) {
      if (!isSupportedTextFile(file.name)) {
        updateRow(file.name, {
          status: 'unsupported',
          message: 'Формат не поддерживается — загрузите .txt или .md',
        })
        continue
      }
      updateRow(file.name, { status: 'processing' })
      try {
        const content = await readTextFile(file)
        await createMutation.mutateAsync({
          title: titleFromFileName(file.name),
          content,
          tags: [],
        })
        updateRow(file.name, { status: 'done', message: 'Добавлен в базу' })
      } catch (e) {
        updateRow(file.name, {
          status: 'error',
          message: (e as Error).message,
        })
      }
    }
  }

  const onDrop = (e: React.DragEvent) => {
    e.preventDefault()
    setDragOver(false)
    void processFiles(Array.from(e.dataTransfer.files))
  }

  return (
    <Box sx={{ maxWidth: 720 }}>
      <PageHeader
        title="Загрузка документов"
        subtitle="Массовое добавление регламентов из текстовых файлов."
      />

      <Alert severity="info" sx={{ mb: 3 }}>
        Поддерживаются только текстовые файлы (.txt, .md). Загрузка PDF/DOCX
        появится после подключения парсинга на backend — сейчас такие файлы
        будут отклонены.
      </Alert>

      <Paper
        variant="outlined"
        onDragOver={(e) => {
          e.preventDefault()
          setDragOver(true)
        }}
        onDragLeave={() => setDragOver(false)}
        onDrop={onDrop}
        sx={{
          p: 5,
          textAlign: 'center',
          borderStyle: 'dashed',
          borderWidth: 2,
          borderColor: dragOver ? 'primary.main' : 'divider',
          bgcolor: dragOver ? 'action.hover' : 'transparent',
          transition: 'all .15s',
        }}
      >
        <CloudUploadIcon sx={{ fontSize: 40, color: 'text.secondary', mb: 1 }} />
        <Typography variant="body1" gutterBottom>
          Перетащите файлы сюда
        </Typography>
        <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
          или выберите вручную
        </Typography>
        <input
          ref={inputRef}
          type="file"
          multiple
          accept=".txt,.md,text/plain,text/markdown"
          hidden
          onChange={(e) => {
            void processFiles(Array.from(e.target.files ?? []))
            e.target.value = ''
          }}
        />
        <Button variant="contained" onClick={() => inputRef.current?.click()}>
          Выбрать файлы
        </Button>
      </Paper>

      {rows.length > 0 && (
        <Box sx={{ mt: 3, display: 'flex', flexDirection: 'column', gap: 1 }}>
          {rows.map((r) => (
            <Paper key={r.name} variant="outlined" sx={{ p: 1.5 }}>
              <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                <Box sx={{ flex: 1, minWidth: 0 }}>
                  <Typography variant="body2" noWrap fontWeight={500}>
                    {r.name}
                  </Typography>
                  {r.status === 'processing' && <LinearProgress sx={{ mt: 1 }} />}
                  {r.message && (
                    <Typography variant="caption" color={r.status === 'error' ? 'error' : 'text.secondary'}>
                      {r.message}
                    </Typography>
                  )}
                </Box>
                <Chip
                  size="small"
                  variant="outlined"
                  label={
                    r.status === 'done'
                      ? 'Готово'
                      : r.status === 'processing'
                        ? 'Загрузка…'
                        : r.status === 'unsupported'
                          ? 'Не поддерживается'
                          : r.status === 'error'
                            ? 'Ошибка'
                            : 'В очереди'
                  }
                  color={r.status === 'done' ? 'success' : r.status === 'error' ? 'error' : 'default'}
                />
              </Box>
            </Paper>
          ))}
        </Box>
      )}
    </Box>
  )
}
