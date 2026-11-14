// Чтение текстовых файлов для загрузки в базу документов.
// Бэкенд сейчас принимает документ как {title, content, tags} —
// PDF/DOCX/XLSX появятся после серверной обработки файлов.

export const SUPPORTED_TEXT_EXTENSIONS = ['.txt', '.md', '.markdown', '.text']

export function isSupportedTextFile(file: File): boolean {
  const name = file.name.toLowerCase()
  return SUPPORTED_TEXT_EXTENSIONS.some((ext) => name.endsWith(ext))
}

export function isSupportedDocumentFile(file: File): boolean {
  const name = file.name.toLowerCase()
  return (
    isSupportedTextFile(file) ||
    name.endsWith('.pdf') ||
    name.endsWith('.docx') ||
    name.endsWith('.xlsx')
  )
}

export function readTextFile(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(String(reader.result ?? ''))
    reader.onerror = () => reject(reader.error ?? new Error('read failed'))
    reader.readAsText(file, 'utf-8')
  })
}

// Заголовок документа из имени файла: убираем расширение.
export function titleFromFileName(fileName: string): string {
  return fileName.replace(/\.[^.]+$/, '')
}
