import Link from "next/link"
import { ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { DEMO_MAILTO } from "@/lib/utils"

export function PlatformHero() {
  return (
    <section className="border-b border-border">
      <div className="mx-auto max-w-7xl px-6 py-16 lg:px-8 lg:py-24">
        <div className="max-w-3xl">
          <p className="text-xs font-medium uppercase tracking-widest text-accent">
            Platform
          </p>
          <h1 className="mt-3 text-balance font-serif text-5xl font-semibold leading-[1.05] tracking-tight text-foreground sm:text-6xl">
            Satu ruang kerja cerdas untuk seluruh praktik hukum.
          </h1>
          <p className="mt-6 text-pretty text-lg leading-relaxed text-muted-foreground">
            MindLaw menyatukan riset, analisis, penyusunan, dan pengetahuan ke
            dalam satu lingkungan aman. Setiap jawaban berlandaskan sumber primer
            dan setiap tindakan tercatat dalam arsip Anda.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <Button render={<a href={DEMO_MAILTO} />} nativeButton={false} size="lg" className="group h-12 px-6 text-base">
              Minta demo
              <ArrowRight className="ml-1 h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Button>
            <Button render={<Link href="/" />} nativeButton={false} size="lg" variant="outline" className="h-12 px-6 text-base">
              Kembali ke ikhtisar
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
