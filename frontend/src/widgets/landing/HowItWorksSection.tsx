import { motion, useReducedMotion } from 'motion/react'
import { landingHowItWorks } from '@/entities/landing/content'
import { fadeUp, staggerContainer, viewportOnce } from '@/shared/lib/motionPresets'

export function HowItWorksSection() {
  const reduceMotion = useReducedMotion()

  return (
    <section
      id={landingHowItWorks.id}
      className="pl-section"
      aria-labelledby="how-heading"
    >
      <div className="pl-container">
        <div className="pl-section__head">
          <p className="pl-label">{'{ '}{landingHowItWorks.label}{' }'}</p>
          <h2 id="how-heading" className="pl-h2">
            {landingHowItWorks.titleLines.map((line) => (
              <span key={line}>{line}</span>
            ))}
          </h2>
        </div>

        <motion.ol
          className="pl-bento"
          initial={reduceMotion ? false : 'hidden'}
          whileInView="visible"
          viewport={viewportOnce}
          variants={staggerContainer}
        >
          {landingHowItWorks.steps.map((step) => (
            <motion.li
              key={step.id}
              variants={fadeUp}
              className={`pl-bento__item pl-bento__item--${step.span} pl-bento--${step.tone}`}
            >
              <p className="pl-bento__num">{step.number}</p>
              <div>
                <h3 className="pl-bento__title">{step.title}</h3>
                <p className="pl-bento__text">{step.description}</p>
              </div>
            </motion.li>
          ))}
        </motion.ol>
      </div>
    </section>
  )
}
