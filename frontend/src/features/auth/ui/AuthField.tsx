import type { ReactNode } from 'react'
import { cn } from '@/shared/lib/cn'

interface Props {
  id: string
  label: string
  error?: string
  // Слот справа от подписи — например, ссылка «Забыли пароль?».
  aside?: ReactNode
  children: ReactNode
}

export function AuthField({ id, label, error, aside, children }: Props) {
  return (
    <div className={cn('pl-field', error && 'pl-field--invalid')}>
      <div className="pl-field__row">
        <label className="pl-field__label" htmlFor={id}>
          {label}
        </label>
        {aside}
      </div>
      {children}
      {error ? (
        <p className="pl-field__error" role="alert">
          {error}
        </p>
      ) : null}
    </div>
  )
}
