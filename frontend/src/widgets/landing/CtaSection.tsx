import { motion, useReducedMotion } from 'motion/react'
import { Link } from 'react-router-dom'
import { landingCta } from '@/entities/landing/content'
import { scrollToHash } from '@/shared/lib/scroll'

export function CtaSection() {
  const reduceMotion = useReducedMotion()

  return (
    <section id={landingCta.id} className="pl-section" aria-labelledby="cta-heading">
      <div className="pl-container">
        <motion.div
          className="pl-cta"
          initial={reduceMotion ? false : { opacity: 0, y: 24 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true, amount: 0.25 }}
          transition={{ duration: 0.7, ease: [0.16, 1, 0.3, 1] }}
        >
          <div className="pl-cta__grid">
            <h2 id="cta-heading" className="pl-cta__title">
              {landingCta.title}
            </h2>
            <div className="pl-cta__side">
              <p className="pl-cta__text">{landingCta.text}</p>
              <div className="pl-cta__actions">
                <a
                  href={landingCta.secondaryCta.href}
                  className="pl-btn pl-cta__ghost"
                  onClick={(event) => {
                    event.preventDefault()
                    scrollToHash(landingCta.secondaryCta.href)
                  }}
                >
                  {landingCta.secondaryCta.label}
                </a>
                <Link to={landingCta.primaryCta.to} className="pl-btn pl-btn--white">
                  {landingCta.primaryCta.label}
                </Link>
              </div>
            </div>
          </div>
        </motion.div>
      </div>
    </section>
  )
}
