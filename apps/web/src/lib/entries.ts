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

// A Tiptap document. Its content is opaque here: Tiptap builds and reads it, and the API
// derives the excerpt and word count from it.
export type TiptapDoc = { type: 'doc'; content?: unknown[] }

export type EntryWithBody = Entry & { body: TiptapDoc }

export type EntryPage = { entries: Entry[]; nextCursor?: string }

export type ListOptions = { cursor?: string; tag?: string; limit?: number }

// Mirrors the API's rules (apps/api/internal/journal/entry_service.go). The API stays the
// authority; these only give the user instant feedback.
export const MOODS = ['happy', 'excited', 'grateful', 'calm', 'neutral', 'tired', 'anxious', 'sad', 'angry'] as const
export type Mood = (typeof MOODS)[number]
export const MAX_TITLE_LENGTH = 200
export const MAX_TAGS = 20
const TAG_PATTERN = /^[a-z0-9][a-z0-9-]{0,31}$/

// What the editor sends. mood '' means "no mood".
export type EntryInput = {
  title: string
  entryDate: string
  mood: Mood | ''
  tags: string[]
  body: TiptapDoc
}

// What actually arrives: Go encodes a nil slice as null, not [].
type RawEntry = Omit<Entry, 'tags'> & { tags: string[] | null }
type RawEntryWithBody = RawEntry & { body: TiptapDoc }
type RawPage = { entries: RawEntry[] | null; nextCursor?: string }

function withBody(raw: RawEntryWithBody): EntryWithBody {
  return { ...raw, tags: raw.tags ?? [] }
}

// The ID comes from the URL, so the user controls it: encode it, so it can only ever be one
// path segment and never something like "../me".
function entryPath(id: string): string {
  return `/api/entries/${encodeURIComponent(id)}`
}

// "  Work Life " -> "work-life". Returns undefined if it still isn't a tag the API accepts.
export function normaliseTag(raw: string): string | undefined {
  const tag = raw.trim().toLowerCase().replace(/\s+/g, '-')
  return TAG_PATTERN.test(tag) ? tag : undefined
}

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

export async function getEntry(id: string): Promise<EntryWithBody> {
  return withBody(await apiFetch<RawEntryWithBody>(entryPath(id)))
}

export async function createEntry(input: EntryInput): Promise<EntryWithBody> {
  return withBody(
    await apiFetch<RawEntryWithBody>('/api/entries', { method: 'POST', body: JSON.stringify(input) }),
  )
}

// PATCH: send only the fields that changed. The API leaves every other field as it is.
export async function updateEntry(id: string, changes: Partial<EntryInput>): Promise<EntryWithBody> {
  return withBody(
    await apiFetch<RawEntryWithBody>(entryPath(id), { method: 'PATCH', body: JSON.stringify(changes) }),
  )
}

// Soft delete: the API sets deletedAt and answers 204 No Content.
export async function deleteEntry(id: string): Promise<void> {
  await apiFetch<void>(entryPath(id), { method: 'DELETE' })
}