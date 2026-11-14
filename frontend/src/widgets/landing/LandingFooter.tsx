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

export function LandingFooter() {
  const year = new Date().getFullYear()

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
          <p>{`© ${year} ${landingFooter.brand}`}</p>
          <p>{landingFooter.brandTagline}</p>
        </div>

        <p className="pl-footer__mark" aria-hidden>
          Физтех.Ответ
        </p>
      </div>
    </footer>
  )
}
