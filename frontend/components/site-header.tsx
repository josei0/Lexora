"use client"

import { useState } from "react"
import { Scale, Menu, X } from "lucide-react"
import { Button } from "@/components/ui/button"
import { DEMO_MAILTO } from "@/lib/utils"

const navLinks = [
  { label: "Platform", href: "/platform" },
  { label: "Kapabilitas", href: "/#capabilities" },
  { label: "Cara Kerja", href: "/#workflow" },
  { label: "Bidang Praktik", href: "/#practice-areas" },
]

export function SiteHeader() {
  const [open, setOpen] = useState(false)

  return (
    <header className="sticky top-0 z-50 border-b border-border/70 bg-background/80 backdrop-blur-md">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6 lg:px-8">
        <a href="/" className="flex items-center gap-2.5">
          <span className="flex h-9 w-9 items-center justify-center rounded-sm bg-primary text-primary-foreground">
            <Scale className="h-5 w-5" aria-hidden="true" />
          </span>
          <span className="font-serif text-xl font-semibold tracking-tight text-foreground">
            MindLaw
          </span>
        </a>

        <nav className="hidden items-center gap-8 md:flex" aria-label="Primary">
          {navLinks.map((link) => (
            <a
              key={link.href}
              href={link.href}
              className="text-sm font-medium text-muted-foreground transition-colors hover:text-foreground"
            >
              {link.label}
            </a>
          ))}
        </nav>

        <div className="hidden items-center gap-3 md:flex">
          <Button render={<a href="/login" />} nativeButton={false} variant="ghost" className="text-sm font-medium">
            Masuk
          </Button>
          <Button render={<a href="/register" />} nativeButton={false} variant="ghost" className="text-sm font-medium">
            Daftar
          </Button>
          <Button render={<a href={DEMO_MAILTO} />} nativeButton={false} className="text-sm font-medium">
            Minta demo
          </Button>
        </div>

        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="inline-flex items-center justify-center rounded-sm p-2 text-foreground md:hidden"
          aria-label={open ? "Tutup menu" : "Buka menu"}
          aria-expanded={open}
        >
          {open ? <X className="h-5 w-5" /> : <Menu className="h-5 w-5" />}
        </button>
      </div>

      {open && (
        <div className="border-t border-border/70 bg-background md:hidden">
          <nav className="mx-auto flex max-w-7xl flex-col gap-1 px-6 py-4" aria-label="Mobile">
            {navLinks.map((link) => (
              <a
                key={link.href}
                href={link.href}
                onClick={() => setOpen(false)}
                className="rounded-sm px-2 py-2 text-sm font-medium text-muted-foreground hover:bg-muted hover:text-foreground"
              >
                {link.label}
              </a>
            ))}
            <div className="mt-3 flex flex-col gap-2">
              <Button render={<a href="/login" />} nativeButton={false} variant="outline" className="w-full">
                Masuk
              </Button>
              <Button render={<a href={DEMO_MAILTO} />} nativeButton={false} className="w-full">
                Minta demo
              </Button>
            </div>
          </nav>
        </div>
      )}
    </header>
  )
}
