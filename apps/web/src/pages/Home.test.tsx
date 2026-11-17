import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError } from '../lib/api'
import { listEntries, type Entry } from '../lib/entries'
import Home from './Home'

vi.mock('../lib/entries', () => ({ listEntries: vi.fn() }))

const entry = (overrides: Partial<Entry> = {}): Entry => ({
  id: 'e1',
  journalId: 'j1',
  title: 'First day at the new job',
  entryDate: '2026-11-03',
  mood: 'excited',
  tags: ['work', 'milestone'],
  excerpt: 'It began with a coffee…',
  wordCount: 412,
  createdAt: '2026-11-03T09:00:00Z',
  updatedAt: '2026-11-03T09:00:00Z',
  ...overrides,
})

function SignInStub() {
  const state = useLocation().state as { from?: string } | null
  return <p>sign in page, came from {state?.from}</p>
}

function renderHome() {
  const router = createMemoryRouter(
    [
      { path: '/', element: <Home /> },
      { path: '/signin', element: <SignInStub /> },
    ],
    { initialEntries: ['/'] },
  )
  render(<RouterProvider router={router} />)
}

describe('Home timeline', () => {
  beforeEach(() => {
    vi.mocked(listEntries).mockReset()
  })

  it('lists entries with their date, mood, word count and tags', async () => {
    vi.mocked(listEntries).mockResolvedValue({ entries: [entry()] })
    renderHome()

    expect(screen.getByRole('status')).toHaveTextContent('Loading your entries')
    expect(await screen.findByRole('heading', { name: 'First day at the new job' })).toBeInTheDocument()
    expect(screen.getByText('It began with a coffee…')).toBeInTheDocument()
    expect(screen.getByText('Mood: excited')).toBeInTheDocument()
    expect(screen.getByText('412 words')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: '#work' })).toBeInTheDocument()
    expect(listEntries).toHaveBeenCalledWith({ tag: undefined, limit: 20 })
  })

  it('says so when there are no entries yet', async () => {
    vi.mocked(listEntries).mockResolvedValue({ entries: [] })
    renderHome()

    expect(await screen.findByText('No entries yet.')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument()
  })

  it('loads the next page with the cursor and hides the button on the last page', async () => {
    vi.mocked(listEntries)
      .mockResolvedValueOnce({ entries: [entry({ id: 'e1', title: 'Newest' })], nextCursor: 'cursor-2' })
      .mockResolvedValueOnce({ entries: [entry({ id: 'e2', title: 'Older' })] })
    const user = userEvent.setup()
    renderHome()

    await user.click(await screen.findByRole('button', { name: 'Load more' }))

    expect(await screen.findByRole('heading', { name: 'Older' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Newest' })).toBeInTheDocument() // appended, not replaced
    expect(listEntries).toHaveBeenLastCalledWith({ tag: undefined, cursor: 'cursor-2', limit: 20 })
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument()
  })

  it('filters by a tag when it is clicked, and shows everything again on "Show all"', async () => {
    vi.mocked(listEntries)
      .mockResolvedValueOnce({ entries: [entry({ id: 'e1', title: 'Tagged', tags: ['work'] })] })
      .mockResolvedValueOnce({ entries: [entry({ id: 'e1', title: 'Tagged', tags: ['work'] })] })
      .mockResolvedValueOnce({ entries: [entry({ id: 'e9', title: 'Everything again', tags: [] })] })
    const user = userEvent.setup()
    renderHome()

    await user.click(await screen.findByRole('button', { name: '#work' }))
    expect(await screen.findByText('#work', { selector: 'strong' })).toBeInTheDocument()
    expect(listEntries).toHaveBeenLastCalledWith({ tag: 'work', limit: 20 })

    await user.click(screen.getByRole('button', { name: 'Show all' }))
    expect(await screen.findByRole('heading', { name: 'Everything again' })).toBeInTheDocument()
    expect(listEntries).toHaveBeenLastCalledWith({ tag: undefined, limit: 20 })
  })

  it('sends the user to sign-in when the API rejects the token', async () => {
    vi.mocked(listEntries).mockImplementation(async () => {
      throw new ApiError(401, 'unauthorized', 'invalid or expired access token')
    })
    renderHome()

    expect(await screen.findByText('sign in page, came from /')).toBeInTheDocument()
  })

  it('shows a friendly error with the request ID for other failures', async () => {
    vi.mocked(listEntries).mockImplementation(async () => {
      throw new ApiError(500, 'internal', 'boom', 'req-123')
    })
    renderHome()

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't load your entries. Please try again. (reference: req-123)",
    )
  })
})