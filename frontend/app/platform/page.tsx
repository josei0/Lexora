import type { Metadata } from "next"
import { SiteHeader } from "@/components/site-header"
import { PlatformHero } from "@/components/platform/platform-hero"
import { PlatformModules } from "@/components/platform/platform-modules"
import { PlatformSecurity } from "@/components/platform/platform-security"
import { CTA } from "@/components/cta"
import { SiteFooter } from "@/components/site-footer"

export const metadata: Metadata = {
  title: "Platform | Lexora",
  description:
    "Jelajahi platform Lexora: riset hukum, analisis kontrak, penyusunan dokumen, dan pustaka pengetahuan privat dalam satu ruang kerja aman untuk firma hukum.",
}

export default function PlatformPage() {
  return (
    <div className="min-h-screen bg-background">
      <SiteHeader />
      <main>
        <PlatformHero />
        <PlatformModules />
        <PlatformSecurity />
        <CTA />
      </main>
      <SiteFooter />
    </div>
  )
}
