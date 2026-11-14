import SearchIcon from '@mui/icons-material/Search'
import { InputAdornment, TextField } from '@mui/material'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'

// Поле поиска каталога документов. Значение живёт в URL (?q=...),
// поэтому ссылку на результаты можно сохранить и переслать.
export function DocumentSearchField() {
  const [params, setParams] = useSearchParams()
  const urlQuery = params.get('q') ?? ''
  const [value, setValue] = useState(urlQuery)

  // Подтягиваем внешние изменения URL (кнопка «сбросить», навигация назад).
  useEffect(() => {
    setValue(urlQuery)
  }, [urlQuery])

  // Debounce: не дёргаем URL на каждое нажатие клавиши.
  useEffect(() => {
    const t = setTimeout(() => {
      setParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          const trimmed = value.trim()
          if (trimmed) next.set('q', trimmed)
          else next.delete('q')
          return next
        },
        { replace: true },
      )
    }, 300)
    return () => clearTimeout(t)
  }, [value, setParams])

  return (
    <TextField
      fullWidth
      size="small"
      placeholder="Поиск по названию, тегам и содержимому…"
      value={value}
      onChange={(e) => setValue(e.target.value)}
      InputProps={{
        startAdornment: (
          <InputAdornment position="start">
            <SearchIcon fontSize="small" />
          </InputAdornment>
        ),
      }}
    />
  )
}
