import { SiteHeader } from "@/components/site-header"
import { Hero } from "@/components/hero"
import { Capabilities } from "@/components/capabilities"
import { Stats } from "@/components/stats"
import { Workflow } from "@/components/workflow"
import { PracticeAreas } from "@/components/practice-areas"
import { CTA } from "@/components/cta"
import { SiteFooter } from "@/components/site-footer"

export default function Page() {
  return (
    <div className="min-h-screen bg-background">
      <SiteHeader />
      <main>
        <Hero />
        <Capabilities />
        <Stats />
        <Workflow />
        <PracticeAreas />
        <CTA />
      </main>
      <SiteFooter />
    </div>
  )
}
