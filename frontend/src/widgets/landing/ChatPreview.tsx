import { landingChatPreview } from '@/entities/landing/content'

// Мокап продукта: диалог с ассистентом — вопрос, инструкция и источники.
export function ChatPreview() {
  return (
    <div className="pl-chat" aria-hidden>
      <div className="pl-chat__frame">
        <aside className="pl-chat__sidebar">
          <p className="pl-chat__brand">
            Физтех<em>.</em>Ответ
          </p>
          <div className="pl-chat__nav">
            {landingChatPreview.nav.map((item, index) => (
              <div
                key={item}
                className={
                  index === 0
                    ? 'pl-chat__nav-item pl-chat__nav-item--active'
                    : 'pl-chat__nav-item'
                }
              >
                <span className="pl-chat__nav-dot" />
                {item}
              </div>
            ))}
          </div>
        </aside>

        <div className="pl-chat__main">
          <div className="pl-chat__head">
            <div>
              <p className="pl-chat__head-title">Новый диалог</p>
              <p className="pl-chat__head-sub">Навигатор по регламентам МФТИ</p>
            </div>
            <span className="pl-source">online</span>
          </div>

          <div className="pl-msg pl-msg--user">{landingChatPreview.question}</div>

          <div className="pl-msg pl-msg--bot">
            <p style={{ margin: 0 }}>{landingChatPreview.answerIntro}</p>
            <ol className="pl-msg__steps">
              {landingChatPreview.steps.map((step, index) => (
                <li key={step}>
                  <span className="pl-msg__step-num">{`0${index + 1}`}</span>
                  <span>{step}</span>
                </li>
              ))}
            </ol>
            <div className="pl-msg__sources">
              {landingChatPreview.sources.map((source) => (
                <span key={source} className="pl-source">
                  <svg viewBox="0 0 12 12" width="10" height="10" fill="none" aria-hidden>
                    <path
                      d="M2.5 1.5h5l2 2v7h-7v-9Z"
                      stroke="currentColor"
                      strokeWidth="1.1"
                      strokeLinejoin="round"
                    />
                  </svg>
                  {source}
                </span>
              ))}
            </div>
          </div>

          <div className="pl-chat__composer">
            <span className="pl-chat__composer-text">{landingChatPreview.composer}</span>
            <span className="pl-chat__composer-btn">
              <svg viewBox="0 0 16 16" width="14" height="14" fill="none" aria-hidden>
                <path
                  d="M8 3v10M3.5 8.5 8 4l4.5 4.5"
                  stroke="currentColor"
                  strokeWidth="1.6"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
