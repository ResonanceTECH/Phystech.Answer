import { motion, useReducedMotion } from 'motion/react'

const SEAL_TEXT = 'ФИЗТЕХ ✦ ОТВЕТ ✦ РЕГЛАМЕНТЫ ✦ ИНСТРУКЦИИ ✦ '

export function PhystechSeal({ className }: { className?: string }) {
  const reduceMotion = useReducedMotion()

  return (
    <motion.div
      className={className}
      aria-hidden
      initial={reduceMotion ? false : { opacity: 0, rotate: -8, scale: 0.92 }}
      whileInView={{ opacity: 1, rotate: 0, scale: 1 }}
      viewport={{ once: true, amount: 0.4 }}
      transition={{ duration: 0.8, ease: [0.16, 1, 0.3, 1] }}
    >
      <div className="pl-seal">
        <svg viewBox="0 0 200 200" className="pl-seal__spin" style={{ position: 'absolute', inset: 0 }}>
          <defs>
            <path
              id="pl-seal-circle"
              d="M100,100 m-72,0 a72,72 0 1,1 144,0 a72,72 0 1,1 -144,0"
            />
          </defs>
          <text className="pl-seal__text">
            <textPath href="#pl-seal-circle" xlinkHref="#pl-seal-circle" startOffset="0%">
              {SEAL_TEXT}
            </textPath>
          </text>
        </svg>
        <div className="pl-seal__core">
          <svg viewBox="0 0 32 32" width="26" height="26" fill="none" aria-hidden>
            <path
              d="M16 4v24M4 16h24M7.5 7.5l17 17M24.5 7.5l-17 17"
              stroke="currentColor"
              strokeWidth="2.2"
              strokeLinecap="round"
            />
          </svg>
        </div>
      </div>
    </motion.div>
  )
}
