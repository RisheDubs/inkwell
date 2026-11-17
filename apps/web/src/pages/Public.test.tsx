import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PublicFrame from '../components/PublicFrame'
import { ApiError } from '../lib/api'
import { getImageUrl } from '../lib/media'
import { getPublicEntry, getPublicImageUrl, getPublicJournal, type PublicJournal as Journal } from '../lib/public'
import PublicEntry from './PublicEntry'
import PublicJournal from './PublicJournal'

vi.mock('../lib/public', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/public')>()),
  getPublicJournal: vi.fn(),
  getPublicEntry: vi.fn(),
  getPublicImageUrl: vi.fn(),
}))
// The private image route must never be used on a public page.
vi.mock('../lib/media', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/media')>()),
  getImageUrl: vi.fn(),
}))

const COVER_ID = '6af4937db325014a79b03c88.png'
const IMAGE_ID = '7bf4937db325014a79b03c99.png'

const summary = (id: string, title: string) => ({
  id,
  title,
  entryDate: '2026-11-15',
  tags: ['work'],
  excerpt: `About ${title}`,
  wordCount: 12,
})

const journal: Journal = {
  handle: 'rishekesh',
  title: 'Rishekesh Journal',
  theme: { paper: 'grid', font: 'mono', accent: '#aa3366' },
  entries: [summary('e1', 'First'), summary('e2', 'Second')],
}

function renderPublic(path: string) {
  const router = createMemoryRouter(
    [
      {
        path: '/j/:handle',
        element: <PublicFrame />,
        children: [
          { index: true, element: <PublicJournal /> },
          { path: ':id', element: <PublicEntry /> },
        ],
      },
    ],
    { initialEntries: [path] },
  )
  render(<RouterProvider router={router} />)
}

describe('public journal page', () => {
  beforeEach(() => {
    vi.mocked(getPublicJournal).mockReset()
    vi.mocked(getPublicEntry).mockReset()
    vi.mocked(getPublicImageUrl).mockReset()
    vi.mocked(getImageUrl).mockReset()
  })

  it('shows the journal, themed, with entries linking to public entry pages', async () => {
    vi.mocked(getPublicJournal).mockResolvedValue(journal)
    renderPublic('/j/rishekesh')

    expect(await screen.findByRole('heading', { level: 1, name: 'Rishekesh Journal' })).toBeInTheDocument()
    expect(getPublicJournal).toHaveBeenCalledWith('rishekesh')
    expect(screen.getByTestId('journal-frame')).toHaveClass('paper-grid')
    expect(screen.getByRole('link', { name: 'First' })).toHaveAttribute('href', '/j/rishekesh/e1')
    expect(screen.queryByRole('button', { name: '#work' })).not.toBeInTheDocument() // tags are labels here
  })

  it.each([
    ['a 404', new ApiError(404, 'not_found', 'not found')],
    ['a server error', new ApiError(500, 'internal', 'boom')],
    ['a network failure', new TypeError('Failed to fetch')],
  ])('shows the same "not available" page for %s', async (_, error) => {
    vi.mocked(getPublicJournal).mockRejectedValue(error)
    renderPublic('/j/someone')

    expect(await screen.findByRole('heading', { name: 'This journal isn’t available' })).toBeInTheDocument()
  })

  it('loads the cover through the public route, never the private one', async () => {
    vi.mocked(getPublicJournal).mockResolvedValue({ ...journal, theme: { ...journal.theme, coverMediaId: COVER_ID } })
    vi.mocked(getPublicImageUrl).mockResolvedValue('https://s3.example/public-cover')
    renderPublic('/j/rishekesh')

    expect(await screen.findByRole('img', { name: 'Journal cover' })).toHaveAttribute('src', 'https://s3.example/public-cover')
    expect(getPublicImageUrl).toHaveBeenCalledWith('rishekesh', COVER_ID)
    expect(getImageUrl).not.toHaveBeenCalled()
  })

  it('loads more entries with the cursor, then hides the button', async () => {
    vi.mocked(getPublicJournal)
      .mockResolvedValueOnce({ ...journal, nextCursor: 'c1' })
      .mockResolvedValueOnce({ ...journal, entries: [summary('e3', 'Third')] })
    const user = userEvent.setup()
    renderPublic('/j/rishekesh')

    await user.click(await screen.findByRole('button', { name: 'Load more' }))

    expect(await screen.findByRole('link', { name: 'Third' })).toBeInTheDocument()
    expect(getPublicJournal).toHaveBeenLastCalledWith('rishekesh', { cursor: 'c1' })
    expect(screen.queryByRole('button', { name: 'Load more' })).not.toBeInTheDocument()
  })
})

describe('public entry page', () => {
  beforeEach(() => {
    vi.mocked(getPublicJournal).mockReset().mockResolvedValue(journal)
    vi.mocked(getPublicEntry).mockReset()
    vi.mocked(getPublicImageUrl).mockReset()
    vi.mocked(getImageUrl).mockReset()
  })

  it('shows the entry read-only, with images through the public route', async () => {
    vi.mocked(getPublicEntry).mockResolvedValue({
      ...summary('e1', 'First'),
      body: {
        type: 'doc',
        content: [
          { type: 'paragraph', content: [{ type: 'text', text: 'Hello from the public page' }] },
          { type: 'mediaImage', attrs: { mediaId: IMAGE_ID, alt: '' } },
        ],
      },
    })
    vi.mocked(getPublicImageUrl).mockResolvedValue('https://s3.example/public-image')
    renderPublic('/j/rishekesh/e1')

    expect(await screen.findByRole('heading', { level: 1, name: 'First' })).toBeInTheDocument()
    expect(getPublicEntry).toHaveBeenCalledWith('rishekesh', 'e1')
    const body = await screen.findByLabelText('Entry text')
    expect(body).toHaveTextContent('Hello from the public page')
    expect(body).toHaveAttribute('contenteditable', 'false')
    expect(await screen.findByRole('img', { name: 'Image in this entry' })).toHaveAttribute('src', 'https://s3.example/public-image')
    expect(getPublicImageUrl).toHaveBeenCalledWith('rishekesh', IMAGE_ID)
    expect(getImageUrl).not.toHaveBeenCalled()
    expect(screen.getByRole('link', { name: '← Rishekesh Journal' })).toHaveAttribute('href', '/j/rishekesh')
  })

  it('shows the same "not available" page for a missing entry', async () => {
    vi.mocked(getPublicEntry).mockRejectedValue(new ApiError(404, 'not_found', 'not found'))
    renderPublic('/j/rishekesh/nope')

    expect(await screen.findByRole('heading', { name: 'This journal isn’t available' })).toBeInTheDocument()
  })
})