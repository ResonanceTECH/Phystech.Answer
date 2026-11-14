import { useState, type InputHTMLAttributes } from 'react'
import { cn } from '@/shared/lib/cn'

interface Props extends InputHTMLAttributes<HTMLInputElement> {
  error?: boolean
}

// Поле пароля с кнопкой показать/скрыть. Все input-пропсы (включая ref из
// react-hook-form) прокидываются как есть.
export function PasswordInput({ className, error, ...props }: Props) {
  const [show, setShow] = useState(false)

  return (
    <div className="pl-pass">
      <input
        {...props}
        type={show ? 'text' : 'password'}
        className={cn('pl-field__input', className)}
        aria-invalid={error || undefined}
      />
      <button
        type="button"
        className="pl-pass__toggle"
        onClick={() => setShow((s) => !s)}
        aria-label={show ? 'Скрыть пароль' : 'Показать пароль'}
        tabIndex={-1}
      >
        {show ? (
          <svg viewBox="0 0 20 20" width="18" height="18" fill="none" aria-hidden>
            <path
              d="M2.7 2.7l14.6 14.6M8.1 8.3a3 3 0 0 0 4.2 4.2M6.4 6.6A8.6 8.6 0 0 0 2 10c1.8 3 4.9 5 8 5a7.6 7.6 0 0 0 3.6-.9M10 5c3.1 0 6.2 2 8 5a11.3 11.3 0 0 1-1.8 2.2"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        ) : (
          <svg viewBox="0 0 20 20" width="18" height="18" fill="none" aria-hidden>
            <path
              d="M2 10c1.8-3 4.9-5 8-5s6.2 2 8 5c-1.8 3-4.9 5-8 5s-6.2-2-8-5Z"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
            <circle cx="10" cy="10" r="2.6" stroke="currentColor" strokeWidth="1.5" />
          </svg>
        )}
      </button>
    </div>
  )
}
