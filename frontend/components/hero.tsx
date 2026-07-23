import Image from "next/image"
import Link from "next/link"
import { ArrowRight, ShieldCheck } from "lucide-react"
import { Button } from "@/components/ui/button"
import { DEMO_MAILTO } from "@/lib/utils"

export function Hero() {
  return (
    <section className="relative overflow-hidden border-b border-border">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="grid items-center gap-12 py-16 lg:grid-cols-2 lg:gap-16 lg:py-24">
          {/* Copy */}
          <div className="max-w-xl">
            <div className="mb-6 inline-flex items-center gap-2 rounded-full border border-border bg-card px-3 py-1">
              <span className="h-1.5 w-1.5 rounded-full bg-accent" aria-hidden="true" />
              <span className="text-xs font-medium uppercase tracking-widest text-muted-foreground">
                Dibuat untuk firma hukum modern
              </span>
            </div>

            <h1 className="text-balance font-serif text-5xl font-semibold leading-[1.05] tracking-tight text-foreground sm:text-6xl lg:text-7xl">
              Presisi dalam hukum, ditenagai kecerdasan.
            </h1>

            <p className="mt-6 text-pretty text-lg leading-relaxed text-muted-foreground">
              Lexora adalah platform hukum cerdas yang membantu advokat meneliti
              peraturan dan putusan, menganalisis kontrak, serta menyusun dokumen
              yang solid, berdasarkan sumber yang dapat diverifikasi dan dirancang
              untuk menjaga kerahasiaan.
            </p>

            <div className="mt-8 flex flex-col gap-3 sm:flex-row">
              <Button render={<a href={DEMO_MAILTO} />} nativeButton={false} size="lg" className="group h-12 px-6 text-base">
                Minta demo
                <ArrowRight className="ml-1 h-4 w-4 transition-transform group-hover:translate-x-0.5" />
              </Button>
              <Button render={<Link href="/platform" />} nativeButton={false} size="lg" variant="outline" className="h-12 px-6 text-base">
                Jelajahi platform
              </Button>
            </div>

            <div className="mt-8 flex items-center gap-2 text-sm text-muted-foreground">
              <ShieldCheck className="h-4 w-4 text-accent" aria-hidden="true" />
              Data terenkripsi. Kerahasiaan advokat-klien terjaga.
            </div>
          </div>

          {/* Image */}
          <div className="relative">
            <div className="relative aspect-[4/5] overflow-hidden rounded-sm border border-border lg:aspect-[5/6]">
              <Image
                src="/hero-law.png"
                alt="Perpustakaan hukum dengan rak-rak tinggi berisi kitab peraturan"
                fill
                priority
                className="object-cover"
                sizes="(max-width: 1024px) 100vw, 50vw"
              />
              <div className="absolute inset-0 bg-gradient-to-t from-primary/40 via-transparent to-transparent" />
            </div>

            <div className="absolute -bottom-6 -left-6 hidden w-64 rounded-sm border border-border bg-card p-5 shadow-xl sm:block">
              <p className="font-serif text-3xl font-semibold text-foreground">2,4 jt+</p>
              <p className="mt-1 text-sm text-muted-foreground">
                Rujukan diverifikasi dari UU, PP, dan putusan pengadilan
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  )
}
