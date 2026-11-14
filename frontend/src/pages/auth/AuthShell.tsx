import { motion, useReducedMotion } from 'motion/react'
import type { ReactNode } from 'react'
import { Link } from 'react-router-dom'
import { paths } from '@/app/router/routePaths'
import { landingHero } from '@/entities/landing/content'
import { easeOutExpo } from '@/shared/lib/motionPresets'
import { HeroBackground } from '@/widgets/landing/HeroBackground'
import '../landing/landing.css'
import './auth.css'

interface Props {
  title: string
  subtitle?: string
  children: ReactNode
}

// Общий каркас auth-страниц: тёмная сцена лендинга, логотип, карточка по центру.
export function AuthShell({ title, subtitle, children }: Props) {
  const reduceMotion = useReducedMotion()

  return (
    <div className="pl-root pl-auth">
      <HeroBackground />

      <header className="pl-auth__top pl-container">
        <Link
          to={paths.home}
          className="pl-logo"
          aria-label={`${landingHero.brand} — ${landingHero.brandTagline}`}
        >
          <span>
            Физтех<em>.</em>Ответ
          </span>
          <small>{landingHero.brandTagline}</small>
        </Link>
        <Link to={paths.home} className="pl-auth__back">
          <svg viewBox="0 0 14 14" width="12" height="12" fill="none" aria-hidden>
            <path
              d="M8.5 2.5 4 7l4.5 4.5M4.5 7H13"
              stroke="currentColor"
              strokeWidth="1.4"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
          На главную
        </Link>
      </header>

      <main className="pl-auth__main pl-container">
        <motion.div
          className="pl-auth__card"
          initial={reduceMotion ? false : { opacity: 0, y: 24 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.7, ease: easeOutExpo }}
        >
          <h1 className="pl-auth__title">{title}</h1>
          {subtitle ? <p className="pl-auth__sub">{subtitle}</p> : null}
          {children}
        </motion.div>
      </main>
    </div>
  )
}
