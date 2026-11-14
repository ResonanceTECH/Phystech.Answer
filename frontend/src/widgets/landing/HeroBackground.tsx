import { motion, useReducedMotion } from 'motion/react'

export function HeroBackground() {
  const reduceMotion = useReducedMotion()

  return (
    <div className="pl-hero-bg" aria-hidden>
      <div className="pl-hero-bg__grid" />
      <motion.div
        className="pl-hero-bg__orb pl-hero-bg__orb--a"
        initial={reduceMotion ? false : { opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 1.2 }}
      />
      <motion.div
        className="pl-hero-bg__orb pl-hero-bg__orb--b"
        initial={reduceMotion ? false : { opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 1.2, delay: 0.15 }}
      />
      <div className="pl-hero-bg__orb pl-hero-bg__orb--c" />
      <motion.div
        className="pl-hero-bg__ring"
        initial={reduceMotion ? false : { opacity: 0, scale: 0.94 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 1.1, ease: [0.16, 1, 0.3, 1] }}
      />
    </div>
  )
}
