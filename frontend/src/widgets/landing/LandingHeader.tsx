import { useEffect, useId, useRef, useState } from 'react'
import { Link } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { landingHero, landingNav } from '@/entities/landing/content'
import { cn } from '@/shared/lib/cn'
import { getActiveSectionId, scrollToHash } from '@/shared/lib/scroll'

const SECTION_IDS = landingNav.map((item) => item.id)

export function LandingHeader() {
  const [scrolled, setScrolled] = useState(false)
  const [activeId, setActiveId] = useState<string | null>(null)
  const [menuOpen, setMenuOpen] = useState(false)
  const menuId = useId()
  const panelRef = useRef<HTMLDivElement>(null)
  const toggleRef = useRef<HTMLButtonElement>(null)

  useEffect(() => {
    let frame = 0
    const onScroll = () => {
      cancelAnimationFrame(frame)
      frame = requestAnimationFrame(() => {
        setScrolled(window.scrollY > 24)
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

  useEffect(() => {
    if (!menuOpen) return

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') setMenuOpen(false)
    }
    function onPointerDown(event: MouseEvent) {
      const target = event.target as Node
      if (panelRef.current?.contains(target) || toggleRef.current?.contains(target)) return
      setMenuOpen(false)
    }

    document.addEventListener('keydown', onKeyDown)
    document.addEventListener('mousedown', onPointerDown)
    document.body.style.overflow = 'hidden'
    return () => {
      document.removeEventListener('keydown', onKeyDown)
      document.removeEventListener('mousedown', onPointerDown)
      document.body.style.overflow = ''
    }
  }, [menuOpen])

  function handleNavClick(href: string) {
    setMenuOpen(false)
    scrollToHash(href)
  }

  return (
    <header className={cn('pl-header', (scrolled || menuOpen) && 'pl-header--solid')}>
      <div className="pl-container pl-header__inner">
        <Link
          to="/"
          className="pl-logo"
          aria-label={`${landingHero.brand} — ${landingHero.brandTagline}`}
        >
          <span>
            Физтех<em>.</em>Ответ
          </span>
          <small>{landingHero.brandTagline}</small>
        </Link>

        <nav className="pl-header__nav" aria-label="Разделы лендинга">
          {landingNav.map((item) => (
            <a
              key={item.id}
              href={item.href}
              onClick={(event) => {
                event.preventDefault()
                handleNavClick(item.href)
              }}
            >
              {item.label}
            </a>
          ))}
        </nav>

        <div className="pl-header__actions">
          <Link to={paths.auth.login} className="pl-btn pl-btn--ghost pl-header__ghost">
            Войти
          </Link>
          <Link to={paths.auth.register} className="pl-btn pl-btn--primary">
            Зарегистрироваться
          </Link>
          <button
            ref={toggleRef}
            type="button"
            className="pl-burger"
            aria-expanded={menuOpen}
            aria-controls={menuId}
            aria-label={menuOpen ? 'Закрыть меню' : 'Открыть меню'}
            onClick={() => setMenuOpen((open) => !open)}
          >
            <svg viewBox="0 0 20 20" width="18" height="18" fill="none" aria-hidden>
              {menuOpen ? (
                <path
                  d="M5 5l10 10M15 5L5 15"
                  stroke="currentColor"
                  strokeWidth="1.6"
                  strokeLinecap="round"
                />
              ) : (
                <path
                  d="M3 6h14M3 10h14M3 14h14"
                  stroke="currentColor"
                  strokeWidth="1.6"
                  strokeLinecap="round"
                />
              )}
            </svg>
          </button>
        </div>
      </div>

      {menuOpen ? (
        <div
          id={menuId}
          ref={panelRef}
          className="pl-mobile-menu"
          role="dialog"
          aria-modal="true"
          aria-label="Мобильное меню"
        >
          <nav aria-label="Мобильные разделы">
            {landingNav.map((item) => (
              <a
                key={item.id}
                href={item.href}
                className={cn(activeId === item.id && 'pl-mobile-menu--active')}
                onClick={(event) => {
                  event.preventDefault()
                  handleNavClick(item.href)
                }}
              >
                {item.label}
              </a>
            ))}
          </nav>
          <div className="pl-mobile-menu__cta">
            <Link
              to={paths.auth.login}
              className="pl-btn pl-btn--ghost"
              onClick={() => setMenuOpen(false)}
            >
              Войти
            </Link>
            <Link
              to={paths.auth.register}
              className="pl-btn pl-btn--primary"
              onClick={() => setMenuOpen(false)}
            >
              Зарегистрироваться
            </Link>
          </div>
        </div>
      ) : null}
    </header>
  )
}
