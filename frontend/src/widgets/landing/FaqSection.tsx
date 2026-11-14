import { useId, useState } from 'react'
import { landingFaq } from '@/entities/landing/content'
import { cn } from '@/shared/lib/cn'

export function FaqSection() {
  const [openId, setOpenId] = useState<string | null>(landingFaq.items[0]?.id ?? null)
  const baseId = useId()

  const mid = Math.ceil(landingFaq.items.length / 2)
  const columns = [landingFaq.items.slice(0, mid), landingFaq.items.slice(mid)]

  return (
    <section id="faq" className="pl-section pl-section--alt" aria-labelledby="faq-heading">
      <div className="pl-container">
        <div className="pl-section__head">
          <p className="pl-label">{'{ '}{landingFaq.label}{' }'}</p>
          <h2 id="faq-heading" className="pl-h2">
            <span>{landingFaq.title}</span>
          </h2>
        </div>

        <div className="pl-faq-grid">
          {columns.map((column, columnIndex) => (
            <div key={columnIndex} className="pl-faq-col">
              {column.map((item) => {
                const panelId = `${baseId}-${item.id}-panel`
                const buttonId = `${baseId}-${item.id}-button`
                const open = openId === item.id

                return (
                  <div key={item.id} className={cn('pl-faq', open && 'pl-faq--open')}>
                    <h3>
                      <button
                        id={buttonId}
                        type="button"
                        className="pl-faq__q"
                        aria-expanded={open}
                        aria-controls={panelId}
                        onClick={() => setOpenId(open ? null : item.id)}
                      >
                        <span>{item.question}</span>
                        <span className="pl-faq__icon" aria-hidden>
                          <svg viewBox="0 0 14 14" width="14" height="14" fill="none">
                            <path
                              d="M7 1v12M1 7h12"
                              stroke="currentColor"
                              strokeWidth="1.8"
                              strokeLinecap="round"
                            />
                          </svg>
                        </span>
                      </button>
                    </h3>
                    <div
                      id={panelId}
                      role="region"
                      aria-labelledby={buttonId}
                      hidden={!open}
                      className="pl-faq__a"
                    >
                      {open ? <p>{item.answer}</p> : null}
                    </div>
                  </div>
                )
              })}
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
