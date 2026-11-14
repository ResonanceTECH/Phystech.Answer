import { motion, useReducedMotion } from 'motion/react'
import { Link } from 'react-router-dom'
import { landingHero } from '@/entities/landing/content'
import { scrollToHash } from '@/shared/lib/scroll'
import { easeOutExpo } from '@/shared/lib/motionPresets'
import { ChatPreview } from '@/widgets/landing/ChatPreview'
import { HeroBackground } from '@/widgets/landing/HeroBackground'

export function HeroSection() {
  const reduceMotion = useReducedMotion()

  return (
    <section className="pl-hero" aria-labelledby="pl-hero-heading">
      <HeroBackground />

      <div className="pl-container pl-hero__inner">
        <div className="pl-hero__grid">
          <div>
            <h1
              id="pl-hero-heading"
              className="pl-display pl-hero__title"
              aria-label={landingHero.lines.join(' ')}
            >
              {landingHero.lines.map((line, index) => (
                <span key={line} className="pl-hero__line">
                  {reduceMotion ? (
                    <span className={index === landingHero.accentLine ? 'pl-accent' : undefined}>
                      {line}
                    </span>
                  ) : (
                    <motion.span
                      className={index === landingHero.accentLine ? 'pl-accent' : undefined}
                      initial={{ y: '0.45em', opacity: 0 }}
                      animate={{ y: '0%', opacity: 1 }}
                      transition={{
                        duration: 0.85,
                        delay: 0.06 + index * 0.09,
                        ease: easeOutExpo,
                      }}
                    >
                      {line}
                    </motion.span>
                  )}
                </span>
              ))}
            </h1>

            <motion.div
              className="pl-hero__actions"
              initial={reduceMotion ? false : { opacity: 0, y: 16 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.65, delay: 0.35, ease: easeOutExpo }}
            >
              <Link to={landingHero.primaryCta.to} className="pl-btn pl-btn--primary">
                {landingHero.primaryCta.label}
              </Link>
              <a
                href={landingHero.secondaryCta.href}
                className="pl-btn pl-btn--ghost"
                onClick={(event) => {
                  event.preventDefault()
                  scrollToHash(landingHero.secondaryCta.href)
                }}
              >
                {landingHero.secondaryCta.label}
              </a>
            </motion.div>
          </div>

          <motion.p
            className="pl-hero__sub"
            initial={reduceMotion ? false : { opacity: 0, y: 18 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.7, delay: 0.45, ease: easeOutExpo }}
          >
            {landingHero.subtitle}
          </motion.p>
        </div>

        <motion.div
          className="pl-hero__foot"
          initial={reduceMotion ? false : { opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.8, delay: 0.6 }}
        >
          <div className="pl-hero__chips">
            {landingHero.suggestions.map((suggestion) => (
              <Link key={suggestion} to="/app" className="pl-chip">
                {suggestion}
              </Link>
            ))}
          </div>

          <button
            type="button"
            className="pl-scroll-cue"
            onClick={() => scrollToHash(landingHero.scrollTarget)}
          >
            <span className="pl-cue-dot" aria-hidden>
              <svg viewBox="0 0 10 14" width="10" height="14" fill="none">
                <path
                  d="M5 1.5 1.75 5.5M5 1.5 8.25 5.5M5 1.5V10"
                  stroke="currentColor"
                  strokeWidth="1.25"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </span>
            {landingHero.scrollLabel}
          </button>
        </motion.div>
      </div>

      <div id="hero-continue" className="pl-hero__peek">
        <div className="pl-container">
          <motion.div
            initial={reduceMotion ? false : { opacity: 0, y: 36 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.9, delay: 0.25, ease: easeOutExpo }}
          >
            <div className="pl-hero__peek-clip">
              <ChatPreview />
              <div className="pl-hero__peek-fade" />
            </div>
          </motion.div>
        </div>
      </div>
    </section>
  )
}
