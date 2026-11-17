// Mirrors handlePattern in apps/api/internal/journal/journal.go. The API stays the authority;
// this only gives a quick, friendly answer before a request.
export const HANDLE_PATTERN = /^[a-z0-9][a-z0-9_-]{2,29}$/

export const HANDLE_RULES =
  'Handles are 3–30 characters: lowercase letters, numbers, - and _, starting with a letter or number.'

// What the user typed, the way the API stores it.
export function normalizeHandle(raw: string): string {
  return raw.trim().toLowerCase()
}

// A message to show, or undefined when the handle looks valid.
export function handleProblem(handle: string): string | undefined {
  return HANDLE_PATTERN.test(handle) ? undefined : HANDLE_RULES
}

export function publicPath(handle: string): string {
  return `/j/${encodeURIComponent(handle)}`
}

// The full link to share, on whatever origin the app is served from (localhost now, CloudFront later).
export function publicUrl(handle: string, origin: string = window.location.origin): string {
  return new URL(publicPath(handle), origin).href
}