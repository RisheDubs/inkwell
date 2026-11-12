import { getAccessToken } from './auth'

// The Go API's failure shape: { "error": { "code", "message", "requestId" } }.
export class ApiError extends Error {
  status: number
  code: string
  requestId?: string

  constructor(status: number, code: string, message: string, requestId?: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.requestId = requestId
  }
}

export type Me = { id: string; email: string }

type ErrorBody = { error?: { code?: string; message?: string; requestId?: string } } | null

// Every call to our API goes through here: token, JSON, and errors handled in one place.
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
  // 1. This function attaches the user's token, so it only ever talks to our own API.
  //    A presigned S3 URL or any other site must never receive it.
  if (!path.startsWith('/api/')) {
    throw new Error(`apiFetch only calls this app's /api/ routes, got: ${path}`)
  }

  // 2. Signed out: fail here, without a pointless request the API would reject anyway.
  const token = await getAccessToken()
  if (!token) {
    throw new ApiError(401, 'unauthorized', 'Not signed in')
  }

  // 3. Keep any headers the caller passed, then add ours.
  const headers = new Headers(init.headers)
  headers.set('Accept', 'application/json')
  headers.set('Authorization', `Bearer ${token}`)
  if (init.body !== undefined && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }

  // 4. Same origin (ADR-0005): Vite's proxy, later CloudFront, forwards /api to the Go API.
  const res = await fetch(path, { ...init, headers })

  // 5. Success with no body, such as DELETE.
  if (res.status === 204) {
    return undefined as T
  }

  // 6. Success with JSON.
  if (res.ok) {
    return (await res.json()) as T
  }

  // 7. Failure: use the API's error shape when there is one. A proxy or load balancer
  //    error page is HTML, so json() fails and we fall back to a generic error.
  const body = (await res.json().catch(() => null)) as ErrorBody
  throw new ApiError(
    res.status,
    body?.error?.code ?? 'http_error',
    body?.error?.message ?? `Request failed with status ${res.status}`,
    body?.error?.requestId,
  )
}

export function getMe(): Promise<Me> {
  return apiFetch<Me>('/api/me')
}