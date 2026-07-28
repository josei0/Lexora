import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

// ponytail: mailto dulu, ganti ke /demo kalau butuh form + CRM
export const DEMO_MAILTO =
  'mailto:halo@mindlaw.web.id?subject=Permintaan%20demo%20MindLaw'
