import { Link } from 'react-router-dom'
import { landingFooter, landingHero } from '@/entities/landing/content'
import { scrollToHash } from '@/shared/lib/scroll'

function FooterLink({ href, label }: { href: string; label: string }) {
  if (href.startsWith('#')) {
    return (
      <a
        href={href}
        onClick={(event) => {
          event.preventDefault()
          scrollToHash(href)
        }}
      >
        {label}
      </a>
    )
  }
  return <Link to={href}>{label}</Link>
}

const CURRENT_YEAR = new Date().getFullYear()

export function LandingFooter() {

  return (
    <footer className="pl-footer">
      <div className="pl-container">
        <div className="pl-footer__cta">
          <h2 className="pl-footer__cta-title">{landingFooter.ctaTitle}</h2>
          <Link to={landingHero.primaryCta.to} className="pl-btn pl-btn--primary">
            {landingHero.primaryCta.label}
          </Link>
        </div>

        <div className="pl-footer__grid">
          <div>
            <p className="pl-logo">
              <span>
                Физтех<em>.</em>Ответ
              </span>
            </p>
            <p className="pl-footer__tagline">{landingFooter.tagline}</p>
          </div>

          {landingFooter.columns.map((column) => (
            <div key={column.title} className="pl-footer__col">
              <p className="pl-footer__col-title">{column.title}</p>
              <ul>
                {column.links.map((link) => (
                  <li key={link.label}>
                    <FooterLink href={link.href} label={link.label} />
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>

        <div className="pl-footer__bottom">
          <p>{`© ${CURRENT_YEAR} ${landingFooter.brand}`}</p>
          <p>{landingFooter.brandTagline}</p>
        </div>

        <div className="pl-footer__mark" aria-hidden>
          <svg viewBox="0 0 1200 150" preserveAspectRatio="xMidYMid meet">
            <defs>
              <linearGradient id="pl-footer-mark-grad" x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor="rgba(167,139,250,0.34)" />
                <stop offset="100%" stopColor="rgba(167,139,250,0.03)" />
              </linearGradient>
            </defs>
            <text
              x="600"
              y="118"
              textAnchor="middle"
              textLength="1150"
              lengthAdjust="spacingAndGlyphs"
              fill="url(#pl-footer-mark-grad)"
            >
              ФИЗТЕХ.ОТВЕТ
            </text>
          </svg>
        </div>
      </div>
    </footer>
  )
}
