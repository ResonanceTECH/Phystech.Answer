import { motion, useReducedMotion } from 'motion/react'
import { Link } from 'react-router-dom'
import { landingCategories } from '@/entities/landing/content'
import { fadeUp, staggerContainer, viewportOnce } from '@/shared/lib/motionPresets'

export function CategoriesSection() {
  const reduceMotion = useReducedMotion()

  return (
    <section
      id={landingCategories.id}
      className="pl-section pl-section--alt"
      aria-labelledby="categories-heading"
    >
      <div className="pl-container">
        <div className="pl-section__head">
          <p className="pl-label">{'{ '}{landingCategories.label}{' }'}</p>
          <h2 id="categories-heading" className="pl-h2">
            {landingCategories.titleLines.map((line) => (
              <span key={line}>{line}</span>
            ))}
          </h2>
          <p className="pl-section__sub">{landingCategories.subtitle}</p>
        </div>

        <motion.div
          className="pl-cloud"
          initial={reduceMotion ? false : 'hidden'}
          whileInView="visible"
          viewport={viewportOnce}
          variants={staggerContainer}
        >
          {landingCategories.items.map((item) => (
            <motion.div key={item.id} variants={fadeUp}>
              <Link to="/app" className={`pl-tag pl-tag--${item.size}`}>
                {item.label}
              </Link>
            </motion.div>
          ))}
        </motion.div>
      </div>
    </section>
  )
}
