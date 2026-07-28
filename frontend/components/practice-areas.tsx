const areas = [
  "Korporasi & M&A",
  "Litigasi",
  "Kekayaan Intelektual",
  "Ketenagakerjaan",
  "Pertanahan & Properti",
  "Kepatuhan & Regulasi",
  "Perpajakan",
  "Pasar Modal",
  "Perlindungan Data Pribadi",
  "Kepailitan & PKPU",
]

export function PracticeAreas() {
  return (
    <section id="practice-areas" className="border-b border-border bg-card/50 py-20 lg:py-28">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="grid gap-12 lg:grid-cols-[1fr_1.2fr] lg:gap-16">
          <div>
            <p className="text-xs font-medium uppercase tracking-widest text-accent">
              Bidang praktik
            </p>
            <h2 className="mt-3 text-balance font-serif text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
              Disetel untuk cara kerja praktik Anda.
            </h2>
            <p className="mt-4 text-pretty text-lg leading-relaxed text-muted-foreground">
              MindLaw menyesuaikan diri dengan terminologi, yurisprudensi, dan
              standar dokumen bidang Anda, sehingga hasilnya terasa seperti dari
              seorang asosiat berpengalaman di tim Anda.
            </p>
          </div>

          <ul className="flex flex-wrap content-start gap-3">
            {areas.map((area) => (
              <li
                key={area}
                className="rounded-full border border-border bg-background px-5 py-2.5 text-sm font-medium text-foreground transition-colors hover:border-accent hover:text-accent"
              >
                {area}
              </li>
            ))}
          </ul>
        </div>
      </div>
    </section>
  )
}
