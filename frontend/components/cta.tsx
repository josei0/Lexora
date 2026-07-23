import { ArrowRight } from "lucide-react"
import { Button } from "@/components/ui/button"
import { DEMO_MAILTO } from "@/lib/utils"

export function CTA() {
  return (
    <section className="border-b border-border py-20 lg:py-28">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="relative overflow-hidden rounded-sm bg-primary px-8 py-16 text-center text-primary-foreground sm:px-16 lg:py-20">
          <h2 className="mx-auto max-w-2xl text-balance font-serif text-4xl font-semibold tracking-tight sm:text-5xl">
            Lihat apa yang bisa Lexora lakukan untuk firma Anda.
          </h2>
          <p className="mx-auto mt-4 max-w-xl text-pretty text-lg leading-relaxed text-primary-foreground/70">
            Jadwalkan sesi peninjauan privat bersama tim solusi hukum kami dan
            jelajahi penerapan yang disesuaikan untuk praktik Anda.
          </p>
          <div className="mt-8 flex flex-col items-center justify-center gap-3 sm:flex-row">
            <Button render={<a href={DEMO_MAILTO} />} nativeButton={false} size="lg" variant="secondary" className="group h-12 px-6 text-base">
              Minta demo
              <ArrowRight className="ml-1 h-4 w-4 transition-transform group-hover:translate-x-0.5" />
            </Button>
            <Button
              render={<a href={DEMO_MAILTO} />}
              nativeButton={false}
              size="lg"
              variant="outline"
              className="h-12 border-primary-foreground/30 bg-transparent px-6 text-base text-primary-foreground hover:bg-primary-foreground/10 hover:text-primary-foreground"
            >
              Hubungi tim penjualan
            </Button>
          </div>
        </div>
      </div>
    </section>
  )
}
