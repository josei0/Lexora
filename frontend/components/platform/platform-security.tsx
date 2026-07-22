import type { LucideIcon } from "lucide-react"
import { ShieldCheck, Lock, FileClock, ServerCog } from "lucide-react"

type Item = {
  icon: LucideIcon
  title: string
  description: string
}

const items: Item[] = [
  {
    icon: ShieldCheck,
    title: "Enkripsi menyeluruh",
    description: "Data terenkripsi saat transit dan saat disimpan, dengan kontrol keamanan yang teruji.",
  },
  {
    icon: Lock,
    title: "Kerahasiaan terjaga",
    description: "Data Anda tidak pernah dipakai untuk pelatihan model dan tetap terisolasi untuk firma Anda.",
  },
  {
    icon: FileClock,
    title: "Jejak audit penuh",
    description: "Setiap kueri, sumber, dan ekspor tercatat untuk kepatuhan dan peninjauan.",
  },
  {
    icon: ServerCog,
    title: "Kontrol enterprise",
    description: "SSO, akses berbasis peran, dan opsi penempatan data untuk firma besar.",
  },
]

export function PlatformSecurity() {
  return (
    <section className="bg-primary py-20 text-primary-foreground lg:py-28">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="max-w-2xl">
          <p className="text-xs font-medium uppercase tracking-widest text-accent">
            Kepercayaan &amp; Keamanan
          </p>
          <h2 className="mt-3 text-balance font-serif text-4xl font-semibold tracking-tight sm:text-5xl">
            Dibangun sesuai standar yang diharapkan klien Anda.
          </h2>
        </div>

        <div className="mt-14 grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {items.map((item) => (
            <div key={item.title} className="border-t border-primary-foreground/20 pt-6">
              <item.icon className="h-6 w-6 text-accent" aria-hidden="true" />
              <h3 className="mt-4 font-serif text-xl font-semibold">{item.title}</h3>
              <p className="mt-2 text-sm leading-relaxed text-primary-foreground/70">
                {item.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
