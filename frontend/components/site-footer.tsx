import { Scale } from "lucide-react"

const columns = [
  {
    heading: "Platform",
    links: ["Riset hukum", "Analisis kontrak", "Penyusunan dokumen", "Keamanan"],
  },
  {
    heading: "Firma",
    links: ["Tentang", "Karier", "Pers", "Kontak"],
  },
  {
    heading: "Sumber daya",
    links: ["Dokumentasi", "Studi kasus", "Webinar", "Pusat kepercayaan"],
  },
  {
    heading: "Legal",
    links: ["Privasi", "Ketentuan", "Kebijakan keamanan", "Kepatuhan"],
  },
]

export function SiteFooter() {
  return (
    <footer className="bg-background">
      <div className="mx-auto max-w-7xl px-6 py-16 lg:px-8">
        <div className="grid gap-12 lg:grid-cols-[1.5fr_2fr]">
          <div className="max-w-sm">
            <a href="#" className="flex items-center gap-2.5">
              <span className="flex h-9 w-9 items-center justify-center rounded-sm bg-primary text-primary-foreground">
                <Scale className="h-5 w-5" aria-hidden="true" />
              </span>
              <span className="font-serif text-xl font-semibold tracking-tight text-foreground">
                Lexora
              </span>
            </a>
            <p className="mt-4 text-sm leading-relaxed text-muted-foreground">
              Kecerdasan hukum modern dan pendamping bagi layanan hukum.
              Berlandaskan sumber yang dapat diverifikasi, dirancang untuk
              menjaga kerahasiaan.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-8 sm:grid-cols-4">
            {columns.map((col) => (
              <div key={col.heading}>
                <h3 className="text-xs font-semibold uppercase tracking-widest text-foreground">
                  {col.heading}
                </h3>
                <ul className="mt-4 space-y-3">
                  {col.links.map((link) => (
                    <li key={link}>
                      <a
                        href="#"
                        className="text-sm text-muted-foreground transition-colors hover:text-foreground"
                      >
                        {link}
                      </a>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>

        <div className="mt-14 flex flex-col items-start justify-between gap-4 border-t border-border pt-8 sm:flex-row sm:items-center">
          <p className="text-sm text-muted-foreground">
            &copy; {new Date().getFullYear()} Lexora. Seluruh hak cipta dilindungi.
          </p>
          <p className="text-xs text-muted-foreground">
            Lexora tidak memberikan nasihat hukum. Ini adalah alat bantu untuk
            profesional hukum berlisensi.
          </p>
        </div>
      </div>
    </footer>
  )
}
