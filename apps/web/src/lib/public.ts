import type { JSONContent } from '@tiptap/react'
import { ApiError } from './api'
import type { Theme } from './journal'
import { isMediaId } from './media'

// Read-only calls for the public page /j/{handle}. Deliberately NOT apiFetch: these must work
// signed out, and must never send the user's token, even when someone is signed in.

// Mirrors journal/public.go. No ownerId, email, journalId, S3 keys or timestamps by design (ADR-0008).
export type PublicEntrySummary = {
  id: string
  title: string
  entryDate: string
  mood?: string
  tags: string[]
  excerpt: string
  wordCount: number
}

export type PublicJournal = {
  handle: string
  title: string
  theme: Theme
  entries: PublicEntrySummary[]
  nextCursor?: string
}

export type PublicEntry = PublicEntrySummary & { body: JSONContent }

type MediaUrl = { url: string; expiresAt: string }
type ErrorBody = { error?: { code?: string; message?: string; requestId?: string } } | null

const REFRESH_BEFORE_MS = 60_000

async function publicFetch<T>(path: string): Promise<T> {
  // 1. Only the public routes. Anything else needs a token and belongs to apiFetch.
  if (!path.startsWith('/api/public/')) {
    throw new Error(`publicFetch only calls /api/public/ routes, got: ${path}`)
  }

  // 2. No Authorization header, and no cookies: the request looks the same as a stranger's.
  const res = await fetch(path, { headers: { Accept: 'application/json' }, credentials: 'omit' })

  if (res.ok) {
    return (await res.json()) as T
  }

  // 3. Same error shape as apiFetch, so the page can treat any failure as "not found".
  const body = (await res.json().catch(() => null)) as ErrorBody
  throw new ApiError(
    res.status,
    body?.error?.code ?? 'http_error',
    body?.error?.message ?? `Request failed with status ${res.status}`,
    body?.error?.requestId,
  )
}

// Callers are async, so a throw here becomes a rejected promise, like any API error.
// One path segment from user-controlled text (the URL bar). Encoding stops "../me" or "a/b"
// from changing which route is called. Empty means there is nothing to find.
function segment(value: string): string {
  const v = value.trim()
  if (!v) throw new ApiError(404, 'not_found', 'not found')
  return encodeURIComponent(v)
}

export async function getPublicJournal(
  handle: string,
  opts: { cursor?: string; tag?: string; limit?: number } = {},
): Promise<PublicJournal> {
  const q = new URLSearchParams()
  if (opts.cursor) q.set('cursor', opts.cursor)
  if (opts.tag) q.set('tag', opts.tag)
  if (opts.limit) q.set('limit', String(opts.limit))
  const qs = q.toString()
  return publicFetch<PublicJournal>(`/api/public/${segment(handle)}${qs ? `?${qs}` : ''}`)
}

export async function getPublicEntry(handle: string, id: string): Promise<PublicEntry> {
  return publicFetch<PublicEntry>(`/api/public/${segment(handle)}/entries/${segment(id)}`)
}

// Separate from media.ts's cache: these links come from a different route and aren't tied to
// whoever is signed in. Key includes the handle, because a media ID only counts under its owner.
const publicUrlCache = new Map<string, Promise<MediaUrl>>()

export async function getPublicImageUrl(
  handle: string,
  mediaId: string,
  now: () => number = Date.now,
): Promise<string> {
  if (!isMediaId(mediaId)) throw new ApiError(404, 'not_found', 'not found')

  const key = `${handle.trim().toLowerCase()}/${mediaId}`
  const cached = publicUrlCache.get(key)
  if (cached) {
    const hit = await cached.catch(() => undefined)
    if (hit && Date.parse(hit.expiresAt) - now() > REFRESH_BEFORE_MS) return hit.url
  }

  const request = publicFetch<MediaUrl>(`/api/public/${segment(handle)}/media/${segment(mediaId)}`)
  publicUrlCache.set(key, request)
  request.catch(() => publicUrlCache.delete(key)) // don't keep failures
  return (await request).url
}

export function clearPublicImageUrlCache(): void {
  publicUrlCache.clear()
}