import Image from "next/image"

const stats = [
  { value: "14 jam", label: "Dihemat tiap advokat per minggu untuk riset dan penyusunan" },
  { value: "99,2%", label: "Akurasi rujukan yang diverifikasi terhadap sumber primer" },
  { value: "60%", label: "Peninjauan kontrak lebih cepat untuk tim bervolume tinggi" },
  { value: "0", label: "Data klien yang digunakan untuk melatih model" },
]

export function Stats() {
  return (
    <section id="platform" className="border-b border-border bg-primary text-primary-foreground">
      <div className="mx-auto grid max-w-7xl items-center gap-12 px-6 py-20 lg:grid-cols-2 lg:gap-16 lg:px-8 lg:py-28">
        <div>
          <p className="text-xs font-medium uppercase tracking-widest text-accent">
            Dalam angka
          </p>
          <h2 className="mt-3 text-balance font-serif text-4xl font-semibold tracking-tight sm:text-5xl">
            Dirancang mengembalikan waktu para advokat.
          </h2>
          <p className="mt-4 text-pretty text-lg leading-relaxed text-primary-foreground/70">
            Firma yang memakai Lexora menghabiskan lebih sedikit waktu untuk
            peninjauan manual dan lebih banyak untuk strategi, nasihat, serta
            pekerjaan yang hanya bisa dilakukan seorang advokat.
          </p>

          <dl className="mt-10 grid grid-cols-2 gap-8">
            {stats.map((stat) => (
              <div key={stat.label}>
                <dt className="font-serif text-4xl font-semibold text-primary-foreground">
                  {stat.value}
                </dt>
                <dd className="mt-2 text-sm leading-relaxed text-primary-foreground/70">
                  {stat.label}
                </dd>
              </div>
            ))}
          </dl>
        </div>

        <div className="relative aspect-[4/3] overflow-hidden rounded-sm border border-primary-foreground/15">
          <Image
            src="/office-detail.png"
            alt="Meja advokat dengan pena dan dokumen hukum"
            fill
            className="object-cover"
            sizes="(max-width: 1024px) 100vw, 50vw"
          />
        </div>
      </div>
    </section>
  )
}
