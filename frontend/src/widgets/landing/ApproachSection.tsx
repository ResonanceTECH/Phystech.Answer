import { motion, useReducedMotion } from 'motion/react'
import { landingApproach } from '@/entities/landing/content'
import { fadeUp, staggerContainer, viewportOnce } from '@/shared/lib/motionPresets'

export function ApproachSection() {
  const reduceMotion = useReducedMotion()

  return (
    <section
      id={landingApproach.id}
      className="pl-section"
      aria-labelledby="approach-heading"
    >
      <div className="pl-container">
        <div className="pl-section__head">
          <p className="pl-label">{'{ '}{landingApproach.label}{' }'}</p>
          <h2 id="approach-heading" className="pl-h2">
            {landingApproach.titleLines.map((line) => (
              <span key={line}>{line}</span>
            ))}
          </h2>
        </div>

        <motion.ul
          className="pl-principles"
          initial={reduceMotion ? false : 'hidden'}
          whileInView="visible"
          viewport={viewportOnce}
          variants={staggerContainer}
        >
          {landingApproach.principles.map((principle) => (
            <motion.li key={principle.id} variants={fadeUp} className="pl-principle">
              <p className="pl-principle__name">{principle.name}</p>
              <div>
                <h3 className="pl-principle__title">{principle.title}</h3>
                <p className="pl-principle__text" style={{ marginTop: '0.875rem' }}>
                  {principle.description}
                </p>
              </div>
            </motion.li>
          ))}
        </motion.ul>
      </div>
    </section>
  )
}
