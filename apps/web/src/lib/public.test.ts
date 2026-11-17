import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from './api'
import { getAccessToken } from './auth'
import { clearPublicImageUrlCache, getPublicEntry, getPublicImageUrl, getPublicJournal } from './public'

// Pretend someone IS signed in, so the tests prove the token is never used anyway.
vi.mock('./auth', () => ({ getAccessToken: vi.fn(async () => 'signed-in-token') }))

// Media ID format is media.ts's business; here only "valid or not" matters.
vi.mock('./media', () => ({ isMediaId: (v: unknown) => v === 'good-id' }))

const fetchMock = vi.fn<typeof fetch>()

function json(status: number, body: unknown): Response {
  return new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
}

function lastCall(): { url: string; init: RequestInit } {
  const [url, init] = fetchMock.mock.calls.at(-1)!
  return { url: String(url), init: init ?? {} }
}

beforeEach(() => {
  vi.stubGlobal('fetch', fetchMock)
  clearPublicImageUrlCache()
})

afterEach(() => {
  fetchMock.mockReset()
  vi.unstubAllGlobals()
})

describe('public fetch', () => {
  it('sends no token and no cookies, even when signed in', async () => {
    fetchMock.mockResolvedValueOnce(json(200, { handle: 'rishekesh', title: 'T', theme: {}, entries: [] }))

    await getPublicJournal('rishekesh')

    const { url, init } = lastCall()
    expect(url).toBe('/api/public/rishekesh')
    expect(new Headers(init.headers).has('Authorization')).toBe(false)
    expect(init.credentials).toBe('omit')
    expect(getAccessToken).not.toHaveBeenCalled()
  })

  it('builds the query string only from options that are set', async () => {
    fetchMock.mockResolvedValueOnce(json(200, { entries: [] }))
    await getPublicJournal('rishekesh', { cursor: 'abc', tag: 'work' })
    expect(lastCall().url).toBe('/api/public/rishekesh?cursor=abc&tag=work')
  })

  it('encodes the handle so it cannot change the route', async () => {
    fetchMock.mockResolvedValueOnce(json(404, { error: { code: 'not_found', message: 'not found' } }))
    await expect(getPublicJournal('../me')).rejects.toBeInstanceOf(ApiError)
    expect(lastCall().url).toBe('/api/public/..%2Fme')
  })

  it('treats an empty handle as not found without calling the API', async () => {
    await expect(getPublicJournal('  ')).rejects.toMatchObject({ status: 404 })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it("turns the API's 404 into an ApiError", async () => {
    fetchMock.mockResolvedValueOnce(
      json(404, { error: { code: 'not_found', message: 'not found', requestId: 'r1' } }),
    )
    await expect(getPublicEntry('rishekesh', 'e1')).rejects.toMatchObject({
      status: 404,
      code: 'not_found',
      requestId: 'r1',
    })
    expect(lastCall().url).toBe('/api/public/rishekesh/entries/e1')
  })

  it('falls back to a generic error when the body is not JSON', async () => {
    fetchMock.mockResolvedValueOnce(new Response('<html>Bad Gateway</html>', { status: 502 }))
    await expect(getPublicJournal('rishekesh')).rejects.toMatchObject({ status: 502, code: 'http_error' })
  })
})

describe('getPublicImageUrl', () => {
  const t0 = Date.parse('2026-11-17T08:00:00Z')
  const fiveMinutesLater = new Date(t0 + 5 * 60_000).toISOString()

  it('rejects a bad media ID without calling the API', async () => {
    await expect(getPublicImageUrl('rishekesh', 'nope')).rejects.toMatchObject({ status: 404 })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('reuses a link that still has time left', async () => {
    fetchMock.mockResolvedValueOnce(json(200, { url: 'https://s3/a', expiresAt: fiveMinutesLater }))

    expect(await getPublicImageUrl('rishekesh', 'good-id', () => t0)).toBe('https://s3/a')
    expect(await getPublicImageUrl('Rishekesh', 'good-id', () => t0 + 60_000)).toBe('https://s3/a')

    expect(fetchMock).toHaveBeenCalledTimes(1)
    expect(lastCall().url).toBe('/api/public/rishekesh/media/good-id')
    expect(new Headers(lastCall().init.headers).has('Authorization')).toBe(false)
  })

  it('fetches a new link when the old one is about to expire', async () => {
    fetchMock
      .mockResolvedValueOnce(json(200, { url: 'https://s3/old', expiresAt: fiveMinutesLater }))
      .mockResolvedValueOnce(json(200, { url: 'https://s3/new', expiresAt: fiveMinutesLater }))

    await getPublicImageUrl('rishekesh', 'good-id', () => t0)
    // 30 seconds left: under the one-minute margin.
    expect(await getPublicImageUrl('rishekesh', 'good-id', () => t0 + 270_000)).toBe('https://s3/new')
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('does not keep failures, so the next call tries again', async () => {
    fetchMock
      .mockResolvedValueOnce(json(404, { error: { code: 'not_found', message: 'not found' } }))
      .mockResolvedValueOnce(json(200, { url: 'https://s3/a', expiresAt: fiveMinutesLater }))

    await expect(getPublicImageUrl('rishekesh', 'good-id', () => t0)).rejects.toBeInstanceOf(ApiError)
    expect(await getPublicImageUrl('rishekesh', 'good-id', () => t0)).toBe('https://s3/a')
  })
})