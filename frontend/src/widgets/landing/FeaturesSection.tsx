import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useState } from 'react'
import { landingFeatures, type FeaturesMode } from '@/entities/landing/content'
import { cn } from '@/shared/lib/cn'
import { easeOutExpo, fadeUp, staggerContainer, viewportOnce } from '@/shared/lib/motionPresets'
import { PhystechSeal } from '@/widgets/landing/PhystechSeal'

export function FeaturesSection() {
  const reduceMotion = useReducedMotion()
  const [mode, setMode] = useState<FeaturesMode>('employee')
  const view = landingFeatures.views[mode]
  const [activeId, setActiveId] = useState(view.steps[0].id)
  const activeIndex = view.steps.findIndex((step) => step.id === activeId)
  const activeStep = view.steps[activeIndex] ?? view.steps[0]

  function handleModeChange(nextMode: FeaturesMode) {
    setMode(nextMode)
    setActiveId(landingFeatures.views[nextMode].steps[0].id)
  }

  return (
    <section id={landingFeatures.id} className="pl-section" aria-labelledby="features-heading">
      <div className="pl-container">
        <div className="pl-features">
          <div className="pl-features__sticky">
            <p className="pl-label">{'{ '}{landingFeatures.label}{' }'}</p>
            <h2 id="features-heading" className="pl-h2" style={{ marginTop: '1.25rem' }}>
              {landingFeatures.titleLines.map((line) => (
                <span key={line}>{line}</span>
              ))}
            </h2>

            <div className="pl-tabs" role="tablist" aria-label="Сценарий работы системы">
              <motion.span
                aria-hidden
                className="pl-tabs__pill"
                initial={false}
                animate={{ x: mode === 'employee' ? 0 : '100%' }}
                transition={
                  reduceMotion
                    ? { duration: 0 }
                    : { type: 'spring', stiffness: 380, damping: 32 }
                }
              />
              {landingFeatures.modes.map((item) => (
                <button
                  key={item.id}
                  type="button"
                  role="tab"
                  aria-selected={mode === item.id}
                  onClick={() => handleModeChange(item.id)}
                  className={cn('pl-tabs__btn', mode === item.id && 'pl-tabs__btn--active')}
                >
                  {item.label}
                </button>
              ))}
            </div>

            <motion.p
              key={mode}
              className="pl-section__sub"
              initial={reduceMotion ? false : { opacity: 0, y: 8 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.35 }}
            >
              {view.subtitle}
            </motion.p>

            <div className="pl-preview">
              <div className="pl-preview__frame">
                <div style={{ position: 'relative' }}>
                  <span className="pl-preview__label">{activeStep.previewLabel}</span>
                  <AnimatePresence mode="wait">
                    <motion.div
                      key={`${mode}-${activeStep.id}`}
                      className="pl-preview__body"
                      initial={reduceMotion ? false : { opacity: 0, y: 12 }}
                      animate={{ opacity: 1, y: 0 }}
                      exit={reduceMotion ? undefined : { opacity: 0, y: -8 }}
                      transition={{ duration: 0.35, ease: easeOutExpo }}
                    >
                      <StepPreview mode={mode} stepId={activeStep.id} />
                    </motion.div>
                  </AnimatePresence>
                </div>
              </div>

              <PhystechSeal
                className="pl-preview__seal"
              />
            </div>
          </div>

          <motion.ol
            key={mode}
            className="pl-steps"
            initial={reduceMotion ? false : 'hidden'}
            whileInView="visible"
            viewport={viewportOnce}
            variants={staggerContainer}
            aria-label={
              mode === 'admin'
                ? 'Шаги работы для администратора'
                : 'Шаги работы для сотрудника'
            }
          >
            <div
              className="pl-steps__progress"
              style={{
                height:
                  activeIndex <= 0
                    ? '0%'
                    : `${(activeIndex / Math.max(view.steps.length - 1, 1)) * 100}%`,
              }}
              aria-hidden
            />
            {view.steps.map((step, index) => {
              const active = step.id === activeId
              return (
                <motion.li
                  key={step.id}
                  variants={fadeUp}
                  className={cn('pl-step', active && 'pl-step--active')}
                >
                  <button
                    type="button"
                    className="pl-step__btn"
                    onClick={() => setActiveId(step.id)}
                    onMouseEnter={() => setActiveId(step.id)}
                    onFocus={() => setActiveId(step.id)}
                    aria-current={active ? 'step' : undefined}
                  >
                    <span className="pl-step__dot" aria-hidden>
                      <span className="pl-step__dot-inner" />
                    </span>
                    <span className="pl-step__row">
                      <span className="pl-step__num">{step.number}</span>
                      <span className="pl-step__body">
                        <span className="pl-step__title">{step.title}</span>
                        <span className="pl-step__text">{step.description}</span>
                        <span className="pl-step__tag">
                          Пример интерфейса · {String(index + 1).padStart(2, '0')}
                        </span>
                      </span>
                    </span>
                  </button>
                </motion.li>
              )
            })}
          </motion.ol>
        </div>
      </div>
    </section>
  )
}

function StepPreview({ mode, stepId }: { mode: FeaturesMode; stepId: string }) {
  if (mode === 'employee') {
    switch (stepId) {
      case 'ask':
        return (
          <div className="pl-mock">
            <div className="pl-mock__msg pl-mock__msg--user">
              Какие документы нужны для поездки на конференцию?
            </div>
            <div className="pl-mock__msg pl-mock__msg--bot">
              Заявка на командировку, приказ и авансовый отчёт после возвращения.
            </div>
          </div>
        )
      case 'categories':
        return (
          <div className="pl-mock">
            {[
              ['Командировки', '6 документов'],
              ['Финансы и ФЛС', '4 документа'],
              ['Подбор персонала', '3 документа'],
            ].map(([title, sub]) => (
              <div key={title} className="pl-mock__row">
                <div>
                  <p className="pl-mock__row-title">{title}</p>
                  <p className="pl-mock__row-sub">{sub}</p>
                </div>
                <span className="pl-mock__badge">Спросить</span>
              </div>
            ))}
          </div>
        )
      case 'sources':
        return (
          <div className="pl-mock">
            <p className="pl-mock__title">Источники ответа</p>
            {[
              ['Положение о командировках', 'п. 3.2'],
              ['Порядок выдачи авансов', 'п. 1.4'],
              ['Форма авансового отчёта', 'приложение 2'],
            ].map(([doc, point]) => (
              <div key={doc} className="pl-mock__row">
                <div>
                  <p className="pl-mock__row-title">{doc}</p>
                  <p className="pl-mock__row-sub">{point}</p>
                </div>
                <span className="pl-mock__badge">Открыть</span>
              </div>
            ))}
          </div>
        )
      default:
        return (
          <div className="pl-mock">
            {[
              ['Командировка на конференцию', 'сегодня'],
              ['Открытие ФЛС для проекта', 'вчера'],
              ['Заявка в ЦИТ на ноутбук', '12 ноября'],
            ].map(([title, sub]) => (
              <div key={title} className="pl-mock__row">
                <div>
                  <p className="pl-mock__row-title">{title}</p>
                  <p className="pl-mock__row-sub">{sub}</p>
                </div>
                <span className="pl-mock__badge">Диалог</span>
              </div>
            ))}
          </div>
        )
    }
  }

  switch (stepId) {
    case 'upload':
      return (
        <div className="pl-mock">
          <div className="pl-mock__head">
            <p className="pl-mock__title">Новый документ</p>
            <span className="pl-mock__badge pl-mock__badge--ok">Загружен</span>
          </div>
          <div className="pl-mock__row">
            <div>
              <p className="pl-mock__row-title">Положение о командировках.pdf</p>
              <p className="pl-mock__row-sub">Регламент · 1.2 МБ</p>
            </div>
          </div>
          <div className="pl-mock__bar">
            <span style={{ width: '100%' }} />
          </div>
        </div>
      )
    case 'sync':
      return (
        <div className="pl-mock">
          <div className="pl-mock__head">
            <p className="pl-mock__title">ML-индекс</p>
            <span className="pl-mock__badge">синхронизация</span>
          </div>
          <div className="pl-mock__row">
            <div>
              <p className="pl-mock__row-title">Чанки регламентов</p>
              <p className="pl-mock__row-sub">эмбеддинги и лексический поиск</p>
            </div>
          </div>
          <div className="pl-mock__bar">
            <span style={{ width: '72%' }} />
          </div>
        </div>
      )
    case 'status':
      return (
        <div className="pl-mock">
          {[
            ['Положение о командировках', 'в индексе'],
            ['Порядок открытия ФЛС', 'в индексе'],
            ['Форма сметы расходов', 'в очереди'],
          ].map(([title, status]) => (
            <div key={title} className="pl-mock__row">
              <div>
                <p className="pl-mock__row-title">{title}</p>
              </div>
              <span
                className={cn(
                  'pl-mock__badge',
                  status === 'в индексе' && 'pl-mock__badge--ok',
                )}
              >
                {status}
              </span>
            </div>
          ))}
        </div>
      )
    default:
      return (
        <div className="pl-mock">
          {[
            ['Положение о командировках', 'ред. 2025'],
            ['Порядок выдачи авансов', 'ред. 2024'],
          ].map(([title, sub]) => (
            <div key={title} className="pl-mock__row">
              <div>
                <p className="pl-mock__row-title">{title}</p>
                <p className="pl-mock__row-sub">{sub}</p>
              </div>
              <span className="pl-mock__badge">Править</span>
            </div>
          ))}
        </div>
      )
  }
}
