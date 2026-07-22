import type { LucideIcon } from "lucide-react"
import { BookMarked, FileCheck2, PenTool, Gavel, Search, ShieldCheck } from "lucide-react"

type Capability = {
  icon: LucideIcon
  title: string
  description: string
}

const capabilities: Capability[] = [
  {
    icon: Search,
    title: "Riset hukum",
    description:
      "Temukan peraturan dan putusan yang relevan dalam hitungan detik, dengan rujukan yang tertaut ke sumber primer.",
  },
  {
    icon: FileCheck2,
    title: "Analisis kontrak",
    description:
      "Tandai klausul berisiko, ketentuan yang hilang, dan penyimpangan dari standar firma sebelum ditandatangani.",
  },
  {
    icon: PenTool,
    title: "Penyusunan dokumen",
    description:
      "Susun memorandum, gugatan, dan perjanjian dari templat firma dengan nada dan presisi yang Anda kendalikan.",
  },
  {
    icon: Gavel,
    title: "Dukungan litigasi",
    description:
      "Bangun kronologi, ringkas berita acara, dan siapkan argumen yang berlandaskan berkas dan hukum.",
  },
  {
    icon: BookMarked,
    title: "Pustaka pengetahuan",
    description:
      "Ubah berkas perkara dan memo Anda menjadi basis pengetahuan privat yang hanya bisa diakses tim Anda.",
  },
  {
    icon: ShieldCheck,
    title: "Privat & aman",
    description:
      "Enkripsi tingkat enterprise, tanpa pelatihan model atas data Anda, dan jejak audit penuh untuk setiap kueri.",
  },
]

export function Capabilities() {
  return (
    <section id="capabilities" className="border-b border-border py-20 lg:py-28">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="max-w-2xl">
          <p className="text-xs font-medium uppercase tracking-widest text-accent">
            Kapabilitas
          </p>
          <h2 className="mt-3 text-balance font-serif text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
            Setiap tahap pekerjaan hukum, ditangani dengan cermat.
          </h2>
          <p className="mt-4 text-pretty text-lg leading-relaxed text-muted-foreground">
            Lexora memperkuat praktik Anda dengan alat yang dirancang khusus untuk
            tuntutan profesional hukum—akurat, terrujuk, dan rahasia.
          </p>
        </div>

        <div className="mt-14 grid gap-px overflow-hidden rounded-sm border border-border bg-border sm:grid-cols-2 lg:grid-cols-3">
          {capabilities.map((item) => (
            <div
              key={item.title}
              className="group bg-card p-8 transition-colors hover:bg-secondary"
            >
              <span className="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-border bg-background text-foreground transition-colors group-hover:border-accent group-hover:text-accent">
                <item.icon className="h-5 w-5" aria-hidden="true" />
              </span>
              <h3 className="mt-5 font-serif text-xl font-semibold text-foreground">
                {item.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                {item.description}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
