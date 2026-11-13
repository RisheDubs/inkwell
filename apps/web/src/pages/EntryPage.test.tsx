import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../lib/api'
import { todayLocal } from '../lib/dates'
import { createEntry, deleteEntry, getEntry, updateEntry, type EntryWithBody } from '../lib/entries'
import EntryPage from './EntryPage'

// Fake the API calls; keep the real MOODS, limits and normaliseTag.
vi.mock('../lib/entries', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/entries')>()),
  getEntry: vi.fn(),
  createEntry: vi.fn(),
  updateEntry: vi.fn(),
  deleteEntry: vi.fn(),
}))

const doc = (text: string) => ({
  type: 'doc' as const,
  content: [{ type: 'paragraph', content: [{ type: 'text', text }] }],
})

const stored = (overrides: Partial<EntryWithBody> = {}): EntryWithBody => ({
  id: 'e1',
  journalId: 'j1',
  title: 'First day',
  entryDate: '2026-11-03',
  mood: 'excited',
  tags: ['work'],
  excerpt: 'It began',
  wordCount: 2,
  createdAt: '2026-11-03T09:00:00Z',
  updatedAt: '2026-11-03T09:00:00Z',
  body: doc('It began'),
  ...overrides,
})

function renderAt(path: string) {
  const router = createMemoryRouter(
    [
      { path: '/entries/new', element: <EntryPage /> },
      { path: '/entries/:id', element: <EntryPage /> },
      { path: '/', element: <p>timeline</p> },
      { path: '/signin', element: <p>sign in page</p> },
    ],
    { initialEntries: [path] },
  )
  render(<RouterProvider router={router} />)
}

describe('EntryPage', () => {
  beforeEach(() => {
    vi.mocked(getEntry).mockReset()
    vi.mocked(createEntry).mockReset()
    vi.mocked(updateEntry).mockReset()
    vi.mocked(deleteEntry).mockReset()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it("starts a new entry on today's date", () => {
    renderAt('/entries/new')
    expect(screen.getByRole('heading', { name: 'New entry' })).toBeInTheDocument()
    expect(screen.getByLabelText('Date')).toHaveValue(todayLocal())
  })

  it('creates an entry from the form, then opens it', async () => {
    vi.mocked(createEntry).mockResolvedValue(stored({ id: 'e9', title: 'Weekend hike' }))
    vi.mocked(getEntry).mockResolvedValue(stored({ id: 'e9', title: 'Weekend hike' }))
    const user = userEvent.setup()
    renderAt('/entries/new')

    await user.type(screen.getByRole('textbox', { name: 'Title' }), '  Weekend hike ')
    await user.clear(screen.getByLabelText('Date'))
    await user.type(screen.getByLabelText('Date'), '2026-11-08')
    await user.selectOptions(screen.getByLabelText('Mood'), 'calm')
    await user.type(screen.getByRole('textbox', { name: 'Tags' }), 'outdoors{Enter}')
    await user.click(screen.getByRole('textbox', { name: 'Entry' }))
    await user.keyboard('Cold air')
    await user.click(screen.getByRole('button', { name: 'Create entry' }))

    expect(createEntry).toHaveBeenCalledWith({
      title: 'Weekend hike',
      entryDate: '2026-11-08',
      mood: 'calm',
      tags: ['outdoors'],
      body: doc('Cold air'),
    })
    expect(await screen.findByRole('heading', { name: 'Edit entry' })).toBeInTheDocument()
    expect(getEntry).toHaveBeenCalledWith('e9')
    expect(screen.getByRole('status')).toHaveTextContent('Entry created.')
  })

  it('loads an entry and sends only the fields that changed', async () => {
    vi.mocked(getEntry).mockResolvedValue(stored())
    vi.mocked(updateEntry).mockResolvedValue(stored({ title: 'First day, renamed' }))
    const user = userEvent.setup()
    renderAt('/entries/e1')

    const title = await screen.findByRole('textbox', { name: 'Title' })
    expect(title).toHaveValue('First day')
    expect(screen.getByRole('textbox', { name: 'Entry' })).toHaveTextContent('It began')

    await user.type(title, ', renamed')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(updateEntry).toHaveBeenCalledWith('e1', { title: 'First day, renamed' })
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
  })

  it('does not call the API when nothing changed', async () => {
    vi.mocked(getEntry).mockResolvedValue(stored())
    const user = userEvent.setup()
    renderAt('/entries/e1')

    await user.click(await screen.findByRole('button', { name: 'Save changes' }))

    expect(updateEntry).not.toHaveBeenCalled()
    expect(screen.getByText('Nothing to save.')).toBeInTheDocument()
  })

  it("shows 'not found' for a missing entry, or one that belongs to someone else", async () => {
    vi.mocked(getEntry).mockImplementation(async () => {
      throw new ApiError(404, 'not_found', 'entry not found')
    })
    renderAt('/entries/someone-elses')

    expect(await screen.findByRole('heading', { name: 'Entry not found' })).toBeInTheDocument()
  })

  it("shows the API's validation message when it rejects a save", async () => {
    vi.mocked(getEntry).mockResolvedValue(stored())
    vi.mocked(updateEntry).mockImplementation(async () => {
      throw new ApiError(400, 'invalid_request', 'entryDate: must not be before 1900-01-01')
    })
    const user = userEvent.setup()
    renderAt('/entries/e1')

    await user.type(await screen.findByRole('textbox', { name: 'Title' }), '!')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('entryDate: must not be before 1900-01-01')
  })

  it('never sends the body when the editor could not display it', async () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {}) // Tiptap warns about the unknown node; expected
    vi.mocked(getEntry).mockResolvedValue(stored({ body: { type: 'doc', content: [{ type: 'mystery' }] } }))
    vi.mocked(updateEntry).mockResolvedValue(stored({ title: 'Kept safe' }))
    const user = userEvent.setup()
    renderAt('/entries/e1')

    expect(await screen.findByText(/the text is locked to keep it safe/)).toBeInTheDocument()

    const title = screen.getByRole('textbox', { name: 'Title' })
    await user.clear(title)
    await user.type(title, 'Kept safe')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(updateEntry).toHaveBeenCalledWith('e1', { title: 'Kept safe' })
  })

  it('offers no delete button for an entry that does not exist yet', () => {
    renderAt('/entries/new')
    expect(screen.queryByRole('button', { name: 'Delete entry' })).not.toBeInTheDocument()
  })

  it('asks before deleting, and does nothing on Cancel', async () => {
    vi.mocked(getEntry).mockResolvedValue(stored())
    const user = userEvent.setup()
    renderAt('/entries/e1')

    await user.click(await screen.findByRole('button', { name: 'Delete entry' }))
    expect(screen.getByRole('group', { name: 'Confirm delete' })).toHaveTextContent('Delete “First day”?')

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(deleteEntry).not.toHaveBeenCalled()
    expect(screen.getByRole('button', { name: 'Delete entry' })).toBeInTheDocument()
  })

  it('deletes after confirmation and goes back to the timeline', async () => {
    vi.mocked(getEntry).mockResolvedValue(stored())
    vi.mocked(deleteEntry).mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderAt('/entries/e1')

    await user.click(await screen.findByRole('button', { name: 'Delete entry' }))
    await user.click(screen.getByRole('button', { name: 'Yes, delete' }))

    expect(deleteEntry).toHaveBeenCalledWith('e1')
    expect(await screen.findByText('timeline')).toBeInTheDocument()
  })

  it('stays on the entry and explains when the delete fails', async () => {
    vi.mocked(getEntry).mockResolvedValue(stored())
    vi.mocked(deleteEntry).mockImplementation(async () => {
      throw new ApiError(500, 'internal', 'boom', 'req-9')
    })
    const user = userEvent.setup()
    renderAt('/entries/e1')

    await user.click(await screen.findByRole('button', { name: 'Delete entry' }))
    await user.click(screen.getByRole('button', { name: 'Yes, delete' }))

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't delete this entry. Please try again. (reference: req-9)",
    )
    expect(screen.getByRole('heading', { name: 'Edit entry' })).toBeInTheDocument()
  })
})