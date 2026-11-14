import { ApproachSection } from '@/widgets/landing/ApproachSection'
import { CategoriesSection } from '@/widgets/landing/CategoriesSection'
import { CtaSection } from '@/widgets/landing/CtaSection'
import { FaqSection } from '@/widgets/landing/FaqSection'
import { FeaturesSection } from '@/widgets/landing/FeaturesSection'
import { HeroSection } from '@/widgets/landing/HeroSection'
import { HowItWorksSection } from '@/widgets/landing/HowItWorksSection'
import { LandingDock } from '@/widgets/landing/LandingDock'
import { LandingFooter } from '@/widgets/landing/LandingFooter'
import { LandingHeader } from '@/widgets/landing/LandingHeader'
import './landing.css'

export function LandingPage() {
  return (
    <div className="pl-root">
      <a href="#landing-main" className="pl-skip">
        Перейти к содержимому
      </a>
      <LandingHeader />
      <LandingDock />
      <main id="landing-main">
        <HeroSection />
        <HowItWorksSection />
        <FeaturesSection />
        <CategoriesSection />
        <ApproachSection />
        <FaqSection />
        <CtaSection />
      </main>
      <LandingFooter />
    </div>
  )
}
