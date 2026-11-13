import { apiFetch } from './api'

// Matches the Go API's Entry JSON (apps/api/internal/journal/entries.go).
export type Entry = {
  id: string
  journalId: string
  title: string
  entryDate: string // "YYYY-MM-DD": the day the entry is about
  mood?: string // the API omits it when empty
  tags: string[]
  excerpt: string
  wordCount: number
  createdAt: string // ISO timestamps
  updatedAt: string
}

export type EntryPage = { entries: Entry[]; nextCursor?: string }

export type ListOptions = { cursor?: string; tag?: string; limit?: number }

// What actually arrives: Go encodes a nil slice as null, not [].
type RawEntry = Omit<Entry, 'tags'> & { tags: string[] | null }
type RawPage = { entries: RawEntry[] | null; nextCursor?: string }

// One page of the user's entries, newest first. The cursor is opaque: pass back exactly what
// the API gave you, never build or decode it, so the API can change its format freely.
export async function listEntries(options: ListOptions = {}): Promise<EntryPage> {
  const params = new URLSearchParams()
  if (options.cursor) params.set('cursor', options.cursor)
  if (options.tag) params.set('tag', options.tag)
  if (options.limit !== undefined) params.set('limit', String(options.limit))

  const query = params.toString()
  const page = await apiFetch<RawPage>(`/api/entries${query ? `?${query}` : ''}`)

  // Normalise once here, so components can always .map() without null checks.
  return {
    entries: (page.entries ?? []).map((entry) => ({ ...entry, tags: entry.tags ?? [] })),
    nextCursor: page.nextCursor || undefined,
  }
}