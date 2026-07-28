const steps = [
  {
    number: "01",
    title: "Tanya dengan bahasa sehari-hari",
    description:
      "Ajukan pertanyaan riset, unggah kontrak, atau jelaskan dokumen yang Anda butuhkan. Tanpa sintaks kueri yang harus dipelajari.",
  },
  {
    number: "02",
    title: "Tinjau jawaban terrujuk",
    description:
      "MindLaw memberi hasil berlandaskan sumber dengan tautan ke peraturan dan putusan, sehingga Anda bisa memverifikasi tiap klaim.",
  },
  {
    number: "03",
    title: "Susun dan sempurnakan",
    description:
      "Buat draf awal dari templat Anda, lalu sunting bersama tim hingga bahasanya benar-benar pas.",
  },
  {
    number: "04",
    title: "Ajukan dengan percaya diri",
    description:
      "Setiap keluaran disertai jejak audit lengkap, siap untuk tinjauan partner, penyampaian ke klien, atau persidangan.",
  },
]

export function Workflow() {
  return (
    <section id="workflow" className="border-b border-border py-20 lg:py-28">
      <div className="mx-auto max-w-7xl px-6 lg:px-8">
        <div className="max-w-2xl">
          <p className="text-xs font-medium uppercase tracking-widest text-accent">
            Cara kerja
          </p>
          <h2 className="mt-3 text-balance font-serif text-4xl font-semibold tracking-tight text-foreground sm:text-5xl">
            Dari pertanyaan hingga pengajuan, dalam empat langkah.
          </h2>
        </div>

        <ol className="mt-14 grid gap-8 md:grid-cols-2 lg:grid-cols-4">
          {steps.map((step) => (
            <li key={step.number} className="relative">
              <span className="font-serif text-5xl font-semibold text-accent/40">
                {step.number}
              </span>
              <div className="mt-4 h-px w-full bg-border" />
              <h3 className="mt-5 font-serif text-xl font-semibold text-foreground">
                {step.title}
              </h3>
              <p className="mt-2 text-sm leading-relaxed text-muted-foreground">
                {step.description}
              </p>
            </li>
          ))}
        </ol>
      </div>
    </section>
  )
}
