import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { landingDock, landingHero } from '@/entities/landing/content'
import { cn } from '@/shared/lib/cn'
import { getActiveSectionId, scrollToHash } from '@/shared/lib/scroll'

const SECTION_IDS = landingDock.map((item) => item.id)

export function LandingDock() {
  const [activeId, setActiveId] = useState<string | null>(null)

  useEffect(() => {
    let frame = 0
    const onScroll = () => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => {
        setActiveId(getActiveSectionId(SECTION_IDS))
      })
    }
    onScroll()
    window.addEventListener('scroll', onScroll, { passive: true })
    return () => {
      cancelAnimationFrame(frame)
      window.removeEventListener('scroll', onScroll)
    }
  }, [])

  return (
    <div className="pl-dock">
      <div className="pl-dock__bar">
        <nav className="pl-dock__nav" aria-label="Разделы лендинга">
          <a
            href="#landing-main"
            className="pl-dock__link"
            onClick={(event) => {
              event.preventDefault()
              window.scrollTo({ top: 0, behavior: 'smooth' })
            }}
          >
            Главная
          </a>
          {landingDock.map((item) => (
            <a
              key={item.id}
              href={item.href}
              className={cn(
                'pl-dock__link',
                activeId === item.id && 'pl-dock__link--active',
              )}
              onClick={(event) => {
                event.preventDefault()
                scrollToHash(item.href)
              }}
            >
              {item.label}
            </a>
          ))}
        </nav>

        <Link to={landingHero.primaryCta.to} className="pl-dock__cta">
          Спросить ассистента
          <span aria-hidden="true">→</span>
        </Link>
      </div>
    </div>
  )
}
