import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiFetch } from './api'
import { listEntries, type Entry } from './entries'

vi.mock('./api', () => ({ apiFetch: vi.fn() }))

const entry = (overrides: Partial<Entry> = {}): Entry => ({
  id: 'e1',
  journalId: 'j1',
  title: 'First day',
  entryDate: '2026-11-03',
  tags: ['work'],
  excerpt: 'It began…',
  wordCount: 412,
  createdAt: '2026-11-03T09:00:00Z',
  updatedAt: '2026-11-03T09:00:00Z',
  ...overrides,
})

beforeEach(() => {
  vi.resetAllMocks()
})

describe('listEntries', () => {
  it('asks for the first page with no query string', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ entries: [] })
    await listEntries()
    expect(apiFetch).toHaveBeenCalledWith('/api/entries')
  })

  it('passes cursor, tag and limit, safely encoded', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ entries: [] })
    await listEntries({ cursor: 'abc=', tag: 'work & life', limit: 20 })
    expect(apiFetch).toHaveBeenCalledWith('/api/entries?cursor=abc%3D&tag=work+%26+life&limit=20')
  })

  it('returns the entries and the cursor for the next page', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ entries: [entry()], nextCursor: 'next-1' })
    await expect(listEntries()).resolves.toEqual({ entries: [entry()], nextCursor: 'next-1' })
  })

  it('turns null tags and null entries from Go into empty arrays', async () => {
    vi.mocked(apiFetch).mockResolvedValueOnce({ entries: [{ ...entry(), tags: null }] })
    expect((await listEntries()).entries[0].tags).toEqual([])

    vi.mocked(apiFetch).mockResolvedValueOnce({ entries: null })
    expect((await listEntries()).entries).toEqual([])
  })

  it('treats a missing cursor as the last page', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ entries: [entry()] })
    expect((await listEntries()).nextCursor).toBeUndefined()
  })
})