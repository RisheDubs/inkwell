import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiFetch } from './api'
import { getJournal, updateJournal } from './journal'

vi.mock('./api', () => ({ apiFetch: vi.fn() }))

beforeEach(() => {
  vi.resetAllMocks()
})

describe('journal API', () => {
  it('loads the signed-in user’s journal', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ id: 'j1' })
    await getJournal()
    expect(apiFetch).toHaveBeenCalledWith('/api/journal')
  })

  it('PATCHes only the changes it is given', async () => {
    vi.mocked(apiFetch).mockResolvedValue({ id: 'j1' })
    await updateJournal({ theme: { accent: '#aa3366' } })
    expect(apiFetch).toHaveBeenCalledWith('/api/journal', {
      method: 'PATCH',
      body: JSON.stringify({ theme: { accent: '#aa3366' } }),
    })
  })
})