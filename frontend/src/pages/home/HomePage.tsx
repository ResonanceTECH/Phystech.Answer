import SendIcon from '@mui/icons-material/Send'
import {
  Box,
  Button,
  Card,
  CardActionArea,
  CardContent,
  Chip,
  Grid,
  InputAdornment,
  Paper,
  TextField,
  Typography,
} from '@mui/material'
import { useState, type FormEvent } from 'react'
import { Link as RouterLink, useNavigate } from 'react-router-dom'
import { useConversationStore } from '@/entities/conversation'
import { useAuthStore } from '@/entities/user'
import { paths } from '@/app/router/routePaths'
import { formatDate } from '@/shared/lib'

// Категории соответствуют реальным документам базы — не показываем
// направления, по которым документов нет.
const CATEGORIES = [
  {
    title: 'Командировки',
    description: 'Оформление поездок, авансы, отчётность',
    question: 'Как оформить командировку для участия в научной конференции?',
  },
  {
    title: 'Техническая поддержка',
    description: 'Оборудование, доступы, заявки в ЦИТ',
    question: 'Сломался рабочий компьютер — куда обращаться?',
  },
  {
    title: 'Подбор персонала',
    description: 'Заявки, согласования, оформление кандидатов',
    question: 'Хочу нанять сотрудника в подразделение — с чего начать?',
  },
  {
    title: 'Финансы и ФЛС',
    description: 'Открытие счетов, реквизиты, договоры',
    question: 'Как открыть финансово-лицевой счёт для научного проекта?',
  },
  {
    title: 'Сметы и бюджет',
    description: 'Формы смет, согласование, исполнение',
    question: 'Как правильно оформить смету расходов по проекту?',
  },
  {
    title: 'Заимствование средств',
    description: 'Внутренние перемещения между подразделениями',
    question: 'Как оформить внутреннее заимствование средств между подразделениями?',
  },
]

const SUGGESTIONS = [
  'Что делать после возвращения из командировки?',
  'Какие документы нужны для поездки на конференцию за границу?',
  'Как подать заявку на подбор преподавателя?',
]

export function HomePage() {
  const [question, setQuestion] = useState('')
  const navigate = useNavigate()
  const conversations = useConversationStore((s) => s.conversations)
  const user = useAuthStore((s) => s.user)

  const ask = (text: string) => {
    const q = text.trim()
    if (!q) return
    navigate(paths.app.chatNew, { state: { question: q } })
  }

  const onSubmit = (e: FormEvent) => {
    e.preventDefault()
    ask(question)
  }

  const recent = conversations.slice(0, 5)

  return (
    <Box sx={{ maxWidth: 960, mx: 'auto' }}>
      <Box sx={{ textAlign: 'center', py: { xs: 4, md: 6 } }}>
        <Typography variant="h4" gutterBottom>
          {user?.firstName
            ? `${user.firstName}, задайте вопрос — получите порядок действий`
            : 'Задайте вопрос — получите порядок действий'}
        </Typography>
        <Typography variant="body1" color="text.secondary" sx={{ maxWidth: 620, mx: 'auto' }}>
          Ассистент найдёт нужные регламенты МФТИ, соберёт инструкцию
          и покажет источники — каждый вывод можно проверить по документу.
        </Typography>
      </Box>

      <Paper
        component="form"
        onSubmit={onSubmit}
        elevation={0}
        sx={{
          p: 1.5,
          display: 'flex',
          gap: 1,
          alignItems: 'center',
          border: '1px solid',
          borderColor: 'divider',
          borderRadius: 3,
          mb: 3,
        }}
      >
        <TextField
          fullWidth
          size="small"
          placeholder="Например: как открыть ФЛС для научного проекта?"
          value={question}
          onChange={(e) => setQuestion(e.target.value)}
          slotProps={{
            input: {
              startAdornment: (
                <InputAdornment position="start">
                  <SendIcon fontSize="small" color="action" />
                </InputAdornment>
              ),
            },
          }}
        />
        <Button type="submit" variant="contained" disabled={!question.trim()}>
          Спросить
        </Button>
      </Paper>

      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 1, justifyContent: 'center', mb: 5 }}>
        {SUGGESTIONS.map((s) => (
          <Chip
            key={s}
            label={s}
            variant="outlined"
            onClick={() => ask(s)}
            sx={{ cursor: 'pointer' }}
          />
        ))}
      </Box>

      <Typography variant="h6" sx={{ mb: 2 }}>
        Категории процедур
      </Typography>
      <Grid container spacing={2} sx={{ mb: 6 }}>
        {CATEGORIES.map((cat) => (
          <Grid key={cat.title} size={{ xs: 12, sm: 6, md: 4 }}>
            <Card variant="outlined" sx={{ height: '100%' }}>
              <CardActionArea
                onClick={() => ask(cat.question)}
                sx={{ height: '100%', alignItems: 'flex-start' }}
              >
                <CardContent>
                  <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 600 }}>
                    {cat.title}
                  </Typography>
                  <Typography variant="body2" color="text.secondary">
                    {cat.description}
                  </Typography>
                </CardContent>
              </CardActionArea>
            </Card>
          </Grid>
        ))}
      </Grid>

      {recent.length > 0 && (
        <>
          <Typography variant="h6" sx={{ mb: 2 }}>
            Последние диалоги
          </Typography>
          <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
            {recent.map((c) => (
              <Card key={c.id} variant="outlined">
                <CardActionArea
                  component={RouterLink}
                  to={paths.app.chat(c.id)}
                  sx={{ px: 2, py: 1.5, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}
                >
                  <Typography variant="body2" noWrap sx={{ flex: 1 }}>
                    {c.title}
                  </Typography>
                  <Typography variant="caption" color="text.secondary" sx={{ ml: 2, flexShrink: 0 }}>
                    {formatDate(new Date(c.updatedAt))}
                  </Typography>
                </CardActionArea>
              </Card>
            ))}
          </Box>
        </>
      )}
    </Box>
  )
}
