import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiFetch } from './api'
import {
  createEntry,
  deleteEntry,
  getEntry,
  listEntries,
  normaliseTag,
  updateEntry,
  type Entry,
  type EntryInput,
} from './entries'

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

const doc = { type: 'doc' as const, content: [{ type: 'paragraph', content: [{ type: 'text', text: 'Hi' }] }] }

const input: EntryInput = { title: 'First day', entryDate: '2026-11-03', mood: 'excited', tags: ['work'], body: doc }

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

describe('getEntry', () => {
  it('loads one entry with its body, normalising null tags', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ ...entry(), tags: null, body: doc })
    await expect(getEntry('e1')).resolves.toEqual({ ...entry(), tags: [], body: doc })
    expect(apiFetch).toHaveBeenCalledWith('/api/entries/e1')
  })

  it('encodes the ID, so a crafted URL cannot reach another API path', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ ...entry(), body: doc })
    await getEntry('../me')
    expect(apiFetch).toHaveBeenCalledWith('/api/entries/..%2Fme')
  })
})

describe('createEntry', () => {
  it('POSTs the entry as JSON', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ ...entry(), body: doc })
    await createEntry(input)
    expect(apiFetch).toHaveBeenCalledWith('/api/entries', { method: 'POST', body: JSON.stringify(input) })
  })
})

describe('updateEntry', () => {
  it('PATCHes only the fields that changed', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ ...entry({ title: 'Renamed' }), body: doc })
    await updateEntry('e1', { title: 'Renamed' })
    expect(apiFetch).toHaveBeenCalledWith('/api/entries/e1', {
      method: 'PATCH',
      body: JSON.stringify({ title: 'Renamed' }),
    })
  })
})

describe('deleteEntry', () => {
  it('sends DELETE for that entry', async () => {
    vi.mocked(apiFetch).mockResolvedValue(undefined)
    await deleteEntry('e1')
    expect(apiFetch).toHaveBeenCalledWith('/api/entries/e1', { method: 'DELETE' })
  })
})

describe('normaliseTag', () => {
  it('lowercases, trims and joins words with hyphens', () => {
    expect(normaliseTag('  Work Life ')).toBe('work-life')
    expect(normaliseTag('day-1')).toBe('day-1')
  })

  it('rejects what the API would reject', () => {
    expect(normaliseTag('café')).toBeUndefined() // only a-z, 0-9 and -
    expect(normaliseTag('-start')).toBeUndefined() // must start with a letter or digit
    expect(normaliseTag('a'.repeat(33))).toBeUndefined() // at most 32 characters
    expect(normaliseTag('   ')).toBeUndefined()
  })
})