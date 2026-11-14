export function scrollToHash(hash: string, behavior: ScrollBehavior = 'smooth') {
  const id = hash.replace(/^#/, '')
  const el = document.getElementById(id)
  if (!el) return
  el.scrollIntoView({ behavior, block: 'start' })
}

export function getActiveSectionId(
  sectionIds: readonly string[],
  offset = 120,
): string | null {
  let active: string | null = null
  for (const id of sectionIds) {
    const el = document.getElementById(id)
    if (!el) continue
    const top = el.getBoundingClientRect().top
    if (top - offset <= 0) active = id
  }
  return active
}
