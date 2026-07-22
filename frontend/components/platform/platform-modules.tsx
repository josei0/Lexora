import Image from "next/image"
import type { LucideIcon } from "lucide-react"
import { Search, FileCheck2, PenTool, BookMarked } from "lucide-react"

type Module = {
  icon: LucideIcon
  eyebrow: string
  title: string
  description: string
  points: string[]
}

const modules: Module[] = [
  {
    icon: Search,
    eyebrow: "Riset",
    title: "Mesin riset hukum",
    description:
      "Ajukan pertanyaan dengan bahasa sehari-hari dan dapatkan jawaban dengan rujukan tepat ke UU, PP, dan putusan pengadilan.",
    points: [
      "Rujukan terlacak ke sumber primer",
      "Filter jenis peraturan dan tanggal",
      "Penanda status peraturan (dicabut/diubah)",
    ],
  },
  {
    icon: FileCheck2,
    eyebrow: "Analisis",
    title: "Analisis kontrak",
    description:
      "Bandingkan perjanjian dengan standar firma, tandai bahasa berisiko, dan hasilkan koreksi yang bisa dipercaya tim Anda.",
    points: [
      "Penilaian risiko per klausul",
      "Deviasi dari standar firma",
      "Ekspor koreksi sekali klik",
    ],
  },
  {
    icon: PenTool,
    eyebrow: "Draf",
    title: "Studio dokumen",
    description:
      "Hasilkan memorandum, gugatan, dan perjanjian dari templat firma sambil menjaga nada, struktur, dan presisi tetap Anda kendalikan.",
    points: [
      "Templat yang disetujui firma",
      "Penyusunan dengan lacak perubahan",
      "Pemeriksaan dasar hukum inline",
    ],
  },
  {
    icon: BookMarked,
    eyebrow: "Pengetahuan",
    title: "Pustaka privat",
    description:
      "Ubah berkas perkara, memo, dan yurisprudensi menjadi basis pengetahuan yang hanya bisa diakses tim Anda.",
    points: [
      "Pengindeksan khusus firma yang aman",
      "Pencarian semantik lintas berkas perkara",
      "Kontrol akses berbasis peran",
    ],
  },
]

export function PlatformModules() {
  return (
    <section className="border-b border-border py-20 lg:py-28">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="max-w-2xl">
          <p className="text-xs font-medium uppercase tracking-widest text-accent">
            Modul
          </p>
          <h2 className="mt-3 text-balance font-serif text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
            Empat modul terhubung, satu sumber kebenaran.
          </h2>
        </div>

        <div className="mt-14 space-y-16 lg:space-y-24">
          {modules.map((mod, i) => (
            <div
              key={mod.title}
              className="grid items-center gap-10 lg:grid-cols-2 lg:gap-16"
            >
              <div className={i % 2 === 1 ? "lg:order-2" : ""}>
                <span className="inline-flex h-11 w-11 items-center justify-center rounded-sm border border-border bg-card text-accent">
                  <mod.icon className="h-5 w-5" aria-hidden="true" />
                </span>
                <p className="mt-5 text-xs font-medium uppercase tracking-widest text-muted-foreground">
                  {mod.eyebrow}
                </p>
                <h3 className="mt-2 font-serif text-3xl font-semibold text-foreground">
                  {mod.title}
                </h3>
                <p className="mt-3 text-pretty leading-relaxed text-muted-foreground">
                  {mod.description}
                </p>
                <ul className="mt-6 space-y-3">
                  {mod.points.map((point) => (
                    <li key={point} className="flex items-start gap-3 text-sm text-foreground">
                      <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full bg-accent" aria-hidden="true" />
                      {point}
                    </li>
                  ))}
                </ul>
              </div>

              <div className={i % 2 === 1 ? "lg:order-1" : ""}>
                <div className="relative aspect-[4/3] overflow-hidden rounded-sm border border-border">
                  <Image
                    src="/office-detail.png"
                    alt=""
                    fill
                    className="object-cover"
                    sizes="(max-width: 1024px) 100vw, 50vw"
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  )
}
