import { apiFetch } from './api'

// Mirrors the API's allow-lists (apps/api/internal/journal/journals.go). The API stays the
// authority; these drive the settings form and the theme checks in lib/theme.ts.
export const PAPERS = ['plain', 'lined', 'dotted', 'grid'] as const
export const FONTS = ['serif', 'sans', 'mono', 'handwriting'] as const
export const ACCENT_PATTERN = /^#[0-9a-f]{6}$/
export const MAX_JOURNAL_TITLE = 100

export type Paper = (typeof PAPERS)[number]
export type Font = (typeof FONTS)[number]
export type Theme = { paper: Paper; font: Font; accent: string }

// GET /api/journal. The response also carries ownerId and coverKey; the frontend has no use for
// either, so the type leaves them out (the cover arrives in step 16c as a media ID).
export type Journal = {
  id: string
  title: string
  visibility: 'private' | 'public'
  theme: Theme
  createdAt: string
}

// PATCH /api/journal: only the fields that changed.
export type JournalChanges = {
  title?: string
  visibility?: 'private' | 'public'
  theme?: Partial<Theme>
}

export function getJournal(): Promise<Journal> {
  return apiFetch<Journal>('/api/journal')
}

export function updateJournal(changes: JournalChanges): Promise<Journal> {
  return apiFetch<Journal>('/api/journal', { method: 'PATCH', body: JSON.stringify(changes) })
}