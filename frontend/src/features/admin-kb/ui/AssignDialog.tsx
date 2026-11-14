import {
  Alert,
  Autocomplete,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  Radio,
  RadioGroup,
  Stack,
  TextField,
  ToggleButton,
  ToggleButtonGroup,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { Document } from '@/entities/document'
import { apiErrorMessage } from '@/shared/api'
import { useAssign, useDepartments, useParticipants } from '../api/adminApi'
import type { AssignRequirement, TargetType } from '../model/types'

interface Props {
  open: boolean
  onClose: () => void
  /** Документы, которые назначаем (сценарий «из списка документов»). */
  documents?: Document[]
  /** Все документы для выбора (сценарий «из карточки сотрудника/отдела»). */
  allDocuments?: Document[]
  /** Предвыбранная цель назначения. */
  target?: { type: TargetType; id?: string }
}

// Назначение документов для ознакомления: всем, подразделению или сотруднику;
// обязательно (со сроком) или как рекомендация для KPI. Сотрудники получают
// уведомление о новом назначении.
export function AssignDialog({ open, onClose, documents, allDocuments, target }: Props) {
  const departments = useDepartments()
  const participants = useParticipants()
  const assign = useAssign()

  const [targetType, setTargetType] = useState<TargetType>(target?.type ?? 'department')
  const [targetId, setTargetId] = useState<string>(target?.id ?? '')
  const [requirement, setRequirement] = useState<AssignRequirement>('required')
  const [dueDate, setDueDate] = useState('')
  const [picked, setPicked] = useState<Document[]>([])

  useEffect(() => {
    if (!open) return
    setTargetType(target?.type ?? 'department')
    setTargetId(target?.id ?? '')
    setRequirement('required')
    setDueDate('')
    setPicked([])
    assign.reset()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open])

  const docs = documents ?? picked
  const employees = (participants.data ?? []).filter((p) => p.active && p.role === 'employee')
  const canSubmit = docs.length > 0 && (targetType === 'all' || targetId !== '') && !assign.isPending

  const submit = () =>
    assign.mutate(
      {
        documentIds: docs.map((d) => d.id),
        targetType,
        targetId: targetType === 'all' ? undefined : targetId,
        requirement,
        dueDate: requirement === 'required' && dueDate ? dueDate : undefined,
      },
      { onSuccess: onClose },
    )

  return (
    <Dialog open={open} onClose={onClose} fullWidth maxWidth="sm">
      <DialogTitle>Назначить для ознакомления</DialogTitle>
      <DialogContent>
        <Stack spacing={2.5} sx={{ pt: 1 }}>
          {documents ? (
            <Typography variant="body2" color="text.secondary">
              {documents.length === 1 ? `Документ: «${documents[0].title}»` : `Документов: ${documents.length}`}
            </Typography>
          ) : (
            <Autocomplete
              multiple
              options={(allDocuments ?? []).filter((d) => (d.visibility ?? 'active') === 'active')}
              getOptionLabel={(d) => d.title}
              value={picked}
              onChange={(_, v) => setPicked(v)}
              renderInput={(params) => <TextField {...params} label="Документы" placeholder="Начните вводить название" />}
            />
          )}

          {!target?.id && (
            <ToggleButtonGroup
              exclusive
              fullWidth
              size="small"
              value={targetType}
              onChange={(_, v: TargetType | null) => {
                if (v) {
                  setTargetType(v)
                  setTargetId('')
                }
              }}
            >
              <ToggleButton value="department">Подразделению</ToggleButton>
              <ToggleButton value="user">Сотруднику</ToggleButton>
              <ToggleButton value="all">Всем</ToggleButton>
            </ToggleButtonGroup>
          )}

          {!target?.id && targetType === 'department' && (
            <Autocomplete
              options={departments.data ?? []}
              getOptionLabel={(d) => d.name}
              value={(departments.data ?? []).find((d) => d.id === targetId) ?? null}
              onChange={(_, v) => setTargetId(v?.id ?? '')}
              renderInput={(params) => <TextField {...params} label="Подразделение" />}
            />
          )}
          {!target?.id && targetType === 'user' && (
            <Autocomplete
              options={employees}
              getOptionLabel={(p) => `${p.name}${p.departmentName ? ` — ${p.departmentName}` : ''}`}
              value={employees.find((p) => p.id === targetId) ?? null}
              onChange={(_, v) => setTargetId(v?.id ?? '')}
              renderInput={(params) => <TextField {...params} label="Сотрудник" />}
            />
          )}

          <RadioGroup row value={requirement} onChange={(e) => setRequirement(e.target.value as AssignRequirement)}>
            <FormControlLabel value="required" control={<Radio />} label="Обязательно" />
            <FormControlLabel value="optional" control={<Radio />} label="Рекомендовано (KPI)" />
          </RadioGroup>

          {requirement === 'required' && (
            <TextField
              type="date"
              label="Срок ознакомления (необязательно)"
              value={dueDate}
              onChange={(e) => setDueDate(e.target.value)}
              slotProps={{ inputLabel: { shrink: true } }}
            />
          )}

          {assign.isError && <Alert severity="error">{apiErrorMessage(assign.error)}</Alert>}
          <Typography variant="caption" color="text.secondary">
            Сотрудники получат уведомление о новом назначении.
          </Typography>
        </Stack>
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>Отмена</Button>
        <Button variant="contained" disabled={!canSubmit} onClick={submit}>
          {assign.isPending ? 'Назначаем…' : 'Назначить'}
        </Button>
      </DialogActions>
    </Dialog>
  )
}
