import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider, useOutletContext } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getMe, type Me } from '../lib/api'
import { getJournal, type Journal } from '../lib/journal'
import { getImageUrl } from '../lib/media'
import { withImages } from '../test/images'
import JournalFrame, { type JournalContext } from './JournalFrame'

vi.mock('../lib/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getMe: vi.fn(),
}))
vi.mock('../lib/journal', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/journal')>()),
  getJournal: vi.fn(),
}))
vi.mock('../lib/media', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/media')>()),
  getImageUrl: vi.fn(),
}))

const me: Me = {
  id: 'u1',
  email: 'rishi@example.com',
  displayName: 'Rishi',
  handle: null,
  aiOptIn: false,
  createdAt: '2026-11-03T09:00:00Z',
}

const journal: Journal = {
  id: 'j1',
  title: 'Rishi’s Journal',
  visibility: 'private',
  theme: { paper: 'grid', font: 'mono', accent: '#aa3366' },
  createdAt: '2026-11-03T09:00:00Z',
}

function TitleProbe() {
  const { journal: current, loadFailed, retry } = useOutletContext<JournalContext>()
  return (
    <>
      <p>journal: {current?.title ?? 'loading'}</p>
      {loadFailed && (
        <button type="button" onClick={retry}>
          retry
        </button>
      )}
    </>
  )
}

function renderFrame(path = '/') {
  const router = createMemoryRouter(
    [{ element: <JournalFrame />, children: [{ path: '/', element: <TitleProbe /> }, { path: '/settings', element: <TitleProbe /> }] }],
    { initialEntries: [path] },
  )
  render(withImages(<RouterProvider router={router} />))
}

describe('JournalFrame', () => {
  beforeEach(() => {
    vi.mocked(getMe).mockReset().mockResolvedValue(me)
    vi.mocked(getJournal).mockReset()
    vi.mocked(getImageUrl).mockReset()
  })

  it('applies the journal’s theme and shares the journal with the page inside', async () => {
    vi.mocked(getJournal).mockResolvedValue(journal)
    renderFrame()

    expect(await screen.findByText('journal: Rishi’s Journal')).toBeInTheDocument()
    const frame = screen.getByTestId('journal-frame')
    expect(frame).toHaveClass('journal', 'paper-grid')
    expect(frame.style.getPropertyValue('--ink-accent')).toBe('#aa3366')
  })

  it('creates the user before asking for the journal, so a first sign-in never gets a 404', async () => {
    vi.mocked(getJournal).mockResolvedValue(journal)
    renderFrame()

    await screen.findByText('journal: Rishi’s Journal')
    expect(vi.mocked(getMe).mock.invocationCallOrder[0]).toBeLessThan(vi.mocked(getJournal).mock.invocationCallOrder[0])
  })

  it('still asks for the journal if /api/me fails', async () => {
    vi.mocked(getMe).mockRejectedValue(new Error('network down'))
    vi.mocked(getJournal).mockResolvedValue(journal)
    renderFrame()

    expect(await screen.findByText('journal: Rishi’s Journal')).toBeInTheDocument()
  })

  it('keeps the default look if the journal cannot be loaded', async () => {
    vi.mocked(getJournal).mockRejectedValue(new Error('network down'))
    renderFrame()

    await waitFor(() => expect(getJournal).toHaveBeenCalled())
    expect(screen.getByTestId('journal-frame')).toHaveClass('paper-dotted')
    expect(screen.getByText('journal: loading')).toBeInTheDocument()
  })

  it('tells the page the load failed, and loads again on retry', async () => {
    vi.mocked(getJournal).mockRejectedValueOnce(new Error('404')).mockResolvedValue(journal)
    const user = userEvent.setup()
    renderFrame()

    await user.click(await screen.findByRole('button', { name: 'retry' }))

    expect(await screen.findByText('journal: Rishi’s Journal')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'retry' })).not.toBeInTheDocument()
  })

  it('shows the cover on the journal home page only', async () => {
    vi.mocked(getJournal).mockResolvedValue({ ...journal, theme: { ...journal.theme, coverMediaId: '6af4937db325014a79b03c88.png' } })
    vi.mocked(getImageUrl).mockResolvedValue('https://s3.example/cover')
    renderFrame('/')

    expect(await screen.findByRole('img', { name: 'Journal cover' })).toHaveAttribute('src', 'https://s3.example/cover')
  })

  it('leaves the cover off other pages, such as settings', async () => {
    vi.mocked(getJournal).mockResolvedValue({ ...journal, theme: { ...journal.theme, coverMediaId: '6af4937db325014a79b03c88.png' } })
    vi.mocked(getImageUrl).mockResolvedValue('https://s3.example/cover')
    renderFrame('/settings')

    expect(await screen.findByText('journal: Rishi’s Journal')).toBeInTheDocument()
    expect(screen.queryByRole('img', { name: 'Journal cover' })).not.toBeInTheDocument()
  })
})