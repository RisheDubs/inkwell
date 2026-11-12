import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, apiFetch, getMe } from './api'
import { getAccessToken } from './auth'

vi.mock('./auth', () => ({ getAccessToken: vi.fn() }))

// A fake fetch, so tests never touch the network.
const fetchMock = vi.fn<typeof fetch>()
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } })
const sentHeaders = () => new Headers(fetchMock.mock.calls[0][1]?.headers)

beforeEach(() => {
  vi.resetAllMocks()
  vi.stubGlobal('fetch', fetchMock)
  vi.mocked(getAccessToken).mockResolvedValue('access-jwt')
})

describe('apiFetch', () => {
  it('sends the access token and returns the parsed JSON', async () => {
    fetchMock.mockResolvedValue(json({ id: 'u1', email: 'ana@example.com' }))

    await expect(apiFetch('/api/me')).resolves.toEqual({ id: 'u1', email: 'ana@example.com' })
    expect(fetchMock.mock.calls[0][0]).toBe('/api/me')
    expect(sentHeaders().get('Authorization')).toBe('Bearer access-jwt')
  })

  it('never sends the token outside our own /api/ paths', async () => {
    await expect(apiFetch('https://evil.example/api/me')).rejects.toThrow()
    await expect(apiFetch('/images/cover.png')).rejects.toThrow()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('does not call the API at all when signed out', async () => {
    vi.mocked(getAccessToken).mockResolvedValue(undefined)

    await expect(apiFetch('/api/me')).rejects.toMatchObject({ status: 401, code: 'unauthorized' })
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('turns the API error shape into an ApiError', async () => {
    fetchMock.mockResolvedValue(
      json({ error: { code: 'not_found', message: 'entry not found', requestId: 'req-1' } }, 404),
    )

    const err = await apiFetch('/api/entries/x').catch((e: unknown) => e)
    expect(err).toBeInstanceOf(ApiError)
    expect(err).toMatchObject({ status: 404, code: 'not_found', message: 'entry not found', requestId: 'req-1' })
  })

  it('copes with an error response that is not JSON, such as a proxy error page', async () => {
    fetchMock.mockResolvedValue(new Response('<html>Bad Gateway</html>', { status: 502 }))

    await expect(apiFetch('/api/me')).rejects.toMatchObject({ status: 502, code: 'http_error' })
  })

  it('marks JSON bodies with a JSON content type and keeps the method', async () => {
    fetchMock.mockResolvedValue(json({ ok: true }))

    await apiFetch('/api/journal', { method: 'PATCH', body: JSON.stringify({ title: 'Mine' }) })
    expect(sentHeaders().get('Content-Type')).toBe('application/json')
    expect(fetchMock.mock.calls[0][1]?.method).toBe('PATCH')
  })

  it('returns undefined for 204 No Content', async () => {
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }))

    await expect(apiFetch('/api/entries/x', { method: 'DELETE' })).resolves.toBeUndefined()
  })
})

describe('getMe', () => {
  it('loads the current user from /api/me', async () => {
    fetchMock.mockResolvedValue(json({ id: 'u1', email: 'ana@example.com' }))

    await expect(getMe()).resolves.toMatchObject({ id: 'u1' })
    expect(fetchMock.mock.calls[0][0]).toBe('/api/me')
  })
})