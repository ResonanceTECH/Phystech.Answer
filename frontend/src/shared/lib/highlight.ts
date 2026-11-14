// Подсветка найденного: слова запроса сводятся к основе (префиксу), чтобы
// находились формы слова — «командировку» → «командиров…» найдёт
// «командировки», «командированию». Номера («794-1», «2.5») ищутся точно.

const STOP = new Set(
  'как что где когда какой какая какие для при над под или and the это этот если чтобы нужно можно ли по на в во из от до за не'.split(' '),
)

function norm(s: string) {
  return s.toLowerCase().replace(/ё/g, 'е')
}

/** Основы слов запроса для поиска форм слова. */
export function queryStems(query: string): string[] {
  const out = new Set<string>()
  for (const raw of norm(query).match(/[\p{L}\p{N}]+(?:[-.][\p{L}\p{N}]+)*/gu) ?? []) {
    if (/\d/.test(raw)) {
      out.add(raw) // номера — точно
      continue
    }
    if (raw.length < 3 || STOP.has(raw)) continue
    // Отбрасываем окончание: для длинных слов — 2–3 буквы, короткие — целиком.
    const stem = raw.length >= 8 ? raw.slice(0, raw.length - 3) : raw.length >= 5 ? raw.slice(0, raw.length - 2) : raw
    out.add(stem)
  }
  return [...out].sort((a, b) => b.length - a.length)
}

function escapeRe(s: string) {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

/** RegExp, находящий слова, начинающиеся с основ запроса (null — нечего искать). */
export function buildMatcher(query: string): RegExp | null {
  const stems = queryStems(query)
  if (stems.length === 0) return null
  // Сопоставляем по нормализованному тексту (ё→е) — длина строки не меняется.
  const alts = stems.map((s) => (/\d/.test(s) ? escapeRe(s) : `${escapeRe(s)}[\\p{L}\\p{N}-]*`))
  return new RegExp(`(?<![\\p{L}\\p{N}])(?:${alts.join('|')})`, 'giu')
}

export interface Segment {
  text: string
  hit: boolean
}

/** Разбивает текст на сегменты «совпадение / не совпадение». */
export function splitByMatches(text: string, re: RegExp | null): Segment[] {
  if (!re || !text) return [{ text, hit: false }]
  const n = norm(text)
  const out: Segment[] = []
  let last = 0
  re.lastIndex = 0
  for (let m = re.exec(n); m; m = re.exec(n)) {
    if (m[0].length === 0) {
      re.lastIndex++
      continue
    }
    if (m.index > last) out.push({ text: text.slice(last, m.index), hit: false })
    out.push({ text: text.slice(m.index, m.index + m[0].length), hit: true })
    last = m.index + m[0].length
  }
  if (last < text.length) out.push({ text: text.slice(last), hit: false })
  return out
}

export interface Snippet {
  segments: Segment[]
  /** Сколько всего совпадений в тексте */
  total: number
}

/**
 * Фрагменты текста вокруг совпадений (для выдачи поиска): до `max` мест,
 * по `radius` символов с каждой стороны, границы по пробелам.
 */
export function snippets(text: string, query: string, max = 2, radius = 90): Snippet | null {
  const re = buildMatcher(query)
  if (!re || !text) return null
  const n = norm(text)
  const hits: number[] = []
  re.lastIndex = 0
  for (let m = re.exec(n); m; m = re.exec(n)) {
    hits.push(m.index)
    if (m[0].length === 0) re.lastIndex++
  }
  if (hits.length === 0) return null
  const parts: Segment[] = []
  let shown = 0
  let lastEnd = -1
  for (const h of hits) {
    if (shown >= max || h < lastEnd) continue
    let from = Math.max(0, h - radius)
    let to = Math.min(text.length, h + radius)
    if (from > 0) from = text.indexOf(' ', from) + 1 || from
    if (to < text.length) to = text.lastIndexOf(' ', to) > h ? text.lastIndexOf(' ', to) : to
    const piece = text.slice(from, to).replace(/\s+/g, ' ')
    if (parts.length > 0) parts.push({ text: ' … ', hit: false })
    else if (from > 0) parts.push({ text: '… ', hit: false })
    parts.push(...splitByMatches(piece, re))
    lastEnd = to
    shown++
  }
  if (lastEnd < text.length) parts.push({ text: ' …', hit: false })
  return { segments: parts, total: hits.length }
}

/**
 * Подсветка в готовом DOM (DOCX/XLSX/распознанный текст): оборачивает
 * совпадения в <mark class="doc-hl">. Возвращает найденные элементы.
 */
export function highlightDom(root: HTMLElement, query: string): HTMLElement[] {
  clearHighlights(root)
  const re = buildMatcher(query)
  if (!re) return []
  const marks: HTMLElement[] = []
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode: (node) =>
      node.nodeValue && node.nodeValue.trim() && !(node.parentElement?.closest('script,style,mark'))
        ? NodeFilter.FILTER_ACCEPT
        : NodeFilter.FILTER_REJECT,
  })
  const nodes: Text[] = []
  for (let n = walker.nextNode(); n; n = walker.nextNode()) nodes.push(n as Text)
  for (const node of nodes) {
    const segs = splitByMatches(node.nodeValue ?? '', re)
    if (!segs.some((s) => s.hit)) continue
    const frag = document.createDocumentFragment()
    for (const s of segs) {
      if (!s.hit) {
        frag.appendChild(document.createTextNode(s.text))
        continue
      }
      const mark = document.createElement('mark')
      mark.className = 'doc-hl'
      mark.textContent = s.text
      marks.push(mark)
      frag.appendChild(mark)
    }
    node.parentNode?.replaceChild(frag, node)
  }
  return marks
}

export function clearHighlights(root: HTMLElement) {
  root.querySelectorAll('mark.doc-hl').forEach((m) => {
    m.replaceWith(document.createTextNode(m.textContent ?? ''))
  })
  root.normalize()
}
