const dateFmt = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
})

const dateTimeFmt = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'long',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
})

const timeFmt = new Intl.DateTimeFormat('ru-RU', {
  hour: '2-digit',
  minute: '2-digit',
})

export function formatDate(iso: string | number | Date): string {
  return dateFmt.format(new Date(iso))
}

export function formatDateTime(iso: string | number | Date): string {
  return dateTimeFmt.format(new Date(iso))
}

export function formatTime(ts: number | Date): string {
  return timeFmt.format(new Date(ts))
}
