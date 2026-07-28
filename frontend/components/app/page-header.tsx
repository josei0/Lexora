// header halaman seragam: judul + subjudul + aksi opsional
export function PageHeader({
  title,
  description,
  action,
}: {
  title: string
  description?: string
  action?: React.ReactNode
}) {
  return (
    <div className="mb-8 flex items-start justify-between gap-4 border-b border-border pb-5">
      <div className="min-w-0">
        <h1 className="font-serif text-3xl leading-tight">{title}</h1>
        {description && <p className="mt-1.5 text-sm text-muted-foreground">{description}</p>}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </div>
  )
}
