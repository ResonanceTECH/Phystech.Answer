import { cn } from '@/shared/lib/cn'
import { passwordScore, passwordScoreLabels } from '../lib/passwordStrength'

// Индикатор надёжности пароля: четыре сегмента + подпись.
export function PasswordStrength({ password }: { password: string }) {
  const score = passwordScore(password)
  return (
    <div className="pl-strength" aria-live="polite">
      <div className="pl-strength__bars" aria-hidden>
        {[1, 2, 3, 4].map((i) => (
          <span
            key={i}
            className={cn('pl-strength__bar', i <= score && 'pl-strength__bar--on')}
          />
        ))}
      </div>
      <span className="pl-strength__label">
        {password ? passwordScoreLabels[score] : 'Минимум 12 символов'}
      </span>
    </div>
  )
}
