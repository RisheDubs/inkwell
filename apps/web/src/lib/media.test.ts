import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { apiFetch } from './api'
import { checkImage, clearImageUrlCache, getImageUrl, isMediaId, uploadImage } from './media'

vi.mock('./api', () => ({ apiFetch: vi.fn() }))

const fetchMock = vi.fn<typeof fetch>()
const MEDIA_ID = '6af4937db325014a79b03c88.png'

const image = (bytes: number, type = 'image/png') => new File([new Uint8Array(bytes)], 'photo', { type })

const ticket = {
  mediaId: MEDIA_ID,
  url: 'https://inkwell-dev-content.s3.ap-southeast-2.amazonaws.com/',
  fields: { key: `users/u1/media/${MEDIA_ID}`, 'Content-Type': 'image/png', policy: 'p', 'X-Amz-Signature': 's' },
  maxBytes: 5 * 1024 * 1024,
  expiresAt: '2026-11-13T12:05:00Z',
}

beforeEach(() => {
  vi.resetAllMocks()
  vi.stubGlobal('fetch', fetchMock)
  clearImageUrlCache()
})
afterEach(() => {
  vi.unstubAllGlobals()
})

describe('checkImage', () => {
  it('accepts the four image types the API allows, up to 5 MB', () => {
    expect(checkImage(image(1024, 'image/jpeg'))).toBeUndefined()
    expect(checkImage(image(5 * 1024 * 1024, 'image/webp'))).toBeUndefined()
  })

  it('rejects other types, including SVG (it can carry script)', () => {
    expect(checkImage(image(10, 'image/svg+xml'))).toBe('Images must be JPEG, PNG, WebP or GIF.')
    expect(checkImage(image(10, 'application/pdf'))).toBe('Images must be JPEG, PNG, WebP or GIF.')
  })

  it('rejects empty files and files over 5 MB', () => {
    expect(checkImage(image(0))).toBe('That file is empty.')
    expect(checkImage(image(5 * 1024 * 1024 + 1))).toBe('Images must be 5 MB or smaller.')
  })
})

describe('uploadImage', () => {
  it('gets a signed form from the API, posts the file to S3 last, and returns the media ID', async () => {
    vi.mocked(apiFetch).mockResolvedValue(ticket)
    fetchMock.mockResolvedValue(new Response(null, { status: 204 }))

    await expect(uploadImage(image(1024))).resolves.toBe(MEDIA_ID)

    expect(apiFetch).toHaveBeenCalledWith('/api/media/uploads', {
      method: 'POST',
      body: JSON.stringify({ contentType: 'image/png' }),
    })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toBe(ticket.url)
    const form = init?.body as FormData
    expect([...form.keys()]).toEqual([...Object.keys(ticket.fields), 'file']) // file last
    expect(new Headers(init?.headers).has('Authorization')).toBe(false) // our token never goes to S3
  })

  it('refuses a bad file without asking the API at all', async () => {
    await expect(uploadImage(image(10, 'image/svg+xml'))).rejects.toThrow('JPEG, PNG, WebP or GIF')
    expect(apiFetch).not.toHaveBeenCalled()
    expect(fetchMock).not.toHaveBeenCalled()
  })

  it('explains when S3 refuses the upload', async () => {
    vi.mocked(apiFetch).mockResolvedValue(ticket)
    fetchMock.mockResolvedValue(
      new Response('<Error><Code>EntityTooLarge</Code></Error>', { status: 400 }),
    )
    await expect(uploadImage(image(1024))).rejects.toThrow('Images must be 5 MB or smaller.')

    fetchMock.mockResolvedValue(new Response('<Error><Code>AccessDenied</Code></Error>', { status: 403 }))
    await expect(uploadImage(image(1024))).rejects.toThrow('The upload was refused (AccessDenied)')
  })
})

describe('getImageUrl', () => {
  const now = Date.parse('2026-11-13T12:00:00Z')
  const link = (url: string, expiresAt: string) => ({ mediaId: MEDIA_ID, url, expiresAt })

  it('asks the API for a short-lived link and reuses it while it is still fresh', async () => {
    vi.mocked(apiFetch).mockResolvedValue(link('https://s3/one', '2026-11-13T12:15:00Z'))

    await expect(getImageUrl(MEDIA_ID, () => now)).resolves.toBe('https://s3/one')
    await expect(getImageUrl(MEDIA_ID, () => now + 5 * 60_000)).resolves.toBe('https://s3/one')

    expect(apiFetch).toHaveBeenCalledOnce()
    expect(apiFetch).toHaveBeenCalledWith(`/api/media/${MEDIA_ID}`)
  })

  it('fetches a new link when the cached one is about to expire', async () => {
    vi.mocked(apiFetch)
      .mockResolvedValueOnce(link('https://s3/one', '2026-11-13T12:15:00Z'))
      .mockResolvedValueOnce(link('https://s3/two', '2026-11-13T12:30:00Z'))

    await getImageUrl(MEDIA_ID, () => now)
    await expect(getImageUrl(MEDIA_ID, () => now + 14.5 * 60_000)).resolves.toBe('https://s3/two')
  })

  it('refuses anything that is not a media ID, without calling the API', async () => {
    await expect(getImageUrl('../me')).rejects.toThrow('Not an image this journal can show.')
    expect(apiFetch).not.toHaveBeenCalled()
    expect(isMediaId('6af4937db325014a79b03c88.svg')).toBe(false)
  })

  it('forgets every link on sign-out', async () => {
    vi.mocked(apiFetch).mockResolvedValue(link('https://s3/one', '2026-11-13T12:15:00Z'))
    await getImageUrl(MEDIA_ID, () => now)

    clearImageUrlCache()
    await getImageUrl(MEDIA_ID, () => now)

    expect(apiFetch).toHaveBeenCalledTimes(2)
  })
})