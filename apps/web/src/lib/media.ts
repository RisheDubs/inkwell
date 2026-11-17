import { apiFetch } from './api'

// Mirrors the API's rules (apps/api/internal/journal/media.go, ADR-0004). S3 enforces them too;
// checking here just tells the user before a wasted upload.
export const IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp', 'image/gif']
export const MAX_IMAGE_BYTES = 5 * 1024 * 1024
const MEDIA_ID = /^[0-9a-f]{24}\.(jpg|png|webp|gif)$/
const REFRESH_BEFORE_MS = 60_000 // fetch a new link when the cached one has under a minute left

type UploadTicket = { mediaId: string; url: string; fields: Record<string, string>; maxBytes: number; expiresAt: string }
type MediaUrl = { mediaId: string; url: string; expiresAt: string }

export class UploadError extends Error {
  constructor(message: string) {
    super(message)
    this.name = 'UploadError'
  }
}

// A media ID is what the entry body stores: "<24 hex>.<ext>". Anything else is refused.
export function isMediaId(value: unknown): value is string {
  return typeof value === 'string' && MEDIA_ID.test(value)
}

// Returns a message for a file S3 would reject, or undefined if it's fine.
export function checkImage(file: File): string | undefined {
  if (!IMAGE_TYPES.includes(file.type)) return 'Images must be JPEG, PNG, WebP or GIF.'
  if (file.size === 0) return 'That file is empty.'
  if (file.size > MAX_IMAGE_BYTES) return 'Images must be 5 MB or smaller.'
  return undefined
}

// Uploads an image straight to S3 and returns its media ID, the only thing the body stores.
export async function uploadImage(file: File): Promise<string> {
  const problem = checkImage(file)
  if (problem) throw new UploadError(problem)

  // 1. Our API signs a one-off upload: this key, this type, at most 5 MB, for 5 minutes.
  const ticket = await apiFetch<UploadTicket>('/api/media/uploads', {
    method: 'POST',
    body: JSON.stringify({ contentType: file.type }),
  })

  // 2. Send the file to S3 as a form. The signed fields go first; S3 ignores anything after the file.
  const form = new FormData()
  for (const [name, value] of Object.entries(ticket.fields)) form.append(name, value)
  form.append('file', file)

  // Plain fetch, never apiFetch: this request goes to S3, which must not see our Cognito token.
  const res = await fetch(ticket.url, { method: 'POST', body: form })
  if (!res.ok) {
    const code = /<Code>([^<]+)<\/Code>/.exec(await res.text().catch(() => ''))?.[1]
    if (code === 'EntityTooLarge') throw new UploadError('Images must be 5 MB or smaller.')
    throw new UploadError(`The upload was refused (${code ?? `status ${res.status}`}). Please try again.`)
  }
  return ticket.mediaId
}

// mediaId -> the API's answer. Holding the promise means two images asking at once share one request.
const urlCache = new Map<string, Promise<MediaUrl>>()

// A short-lived link to show an image. Reused until it has under a minute left.
export async function getImageUrl(mediaId: string, now: () => number = Date.now): Promise<string> {
  if (!isMediaId(mediaId)) throw new UploadError('Not an image this journal can show.')

  const cached = urlCache.get(mediaId)
  if (cached) {
    const hit = await cached.catch(() => undefined)
    if (hit && Date.parse(hit.expiresAt) - now() > REFRESH_BEFORE_MS) return hit.url
  }

  const request = apiFetch<MediaUrl>(`/api/media/${encodeURIComponent(mediaId)}`)
  urlCache.set(mediaId, request)
  request.catch(() => urlCache.delete(mediaId)) // don't keep failures
  return (await request).url
}

// Links are signed for one user. Call on sign-out, so the next person on this tab never gets them.
export function clearImageUrlCache(): void {
  urlCache.clear()
}