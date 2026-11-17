import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, getMe, updateMe, type Me } from '../lib/api'
import { updateJournal, type Journal } from '../lib/journal'
import SharingSettings from './SharingSettings'

vi.mock('../lib/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getMe: vi.fn(),
  updateMe: vi.fn(),
}))
vi.mock('../lib/journal', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/journal')>()),
  updateJournal: vi.fn(),
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
  title: 'Rishekesh Journal',
  visibility: 'private',
  theme: { paper: 'dotted', font: 'serif', accent: '#2f5d50' },
  createdAt: '2026-11-03T09:00:00Z',
}

// Holds the journal like Settings does, so a save shows up in the next render.
function Harness({ initial }: { initial: Journal }) {
  const [current, setCurrent] = useState(initial)
  return (
    <>
      <p>visibility: {current.visibility}</p>
      <SharingSettings journal={current} onSaved={setCurrent} />
    </>
  )
}

function renderSharing(initial: Journal = journal) {
  const router = createMemoryRouter([{ path: '/settings', element: <Harness initial={initial} /> }], {
    initialEntries: ['/settings'],
  })
  render(<RouterProvider router={router} />)
}

describe('SharingSettings', () => {
  beforeEach(() => {
    vi.mocked(getMe).mockReset().mockResolvedValue(me)
    vi.mocked(updateMe).mockReset()
    vi.mocked(updateJournal).mockReset()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('needs a handle before the journal can be made public', async () => {
    renderSharing()

    expect(await screen.findByRole('textbox', { name: 'Handle' })).toHaveValue('')
    expect(screen.getByRole('checkbox', { name: 'Share this journal publicly' })).toBeDisabled()
    expect(screen.getByText('Choose a handle first.')).toBeInTheDocument()
  })

  it('saves the handle the way the API stores it', async () => {
    vi.mocked(updateMe).mockResolvedValue({ ...me, handle: 'rishekesh' })
    const user = userEvent.setup()
    renderSharing()

    await user.type(await screen.findByRole('textbox', { name: 'Handle' }), '  Rishekesh ')
    await user.click(screen.getByRole('button', { name: 'Save handle' }))

    expect(updateMe).toHaveBeenCalledWith({ handle: 'rishekesh' })
    expect(await screen.findByText('Handle saved.')).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Share this journal publicly' })).toBeEnabled()
  })

  it('explains a malformed handle without calling the API', async () => {
    const user = userEvent.setup()
    renderSharing()

    await user.type(await screen.findByRole('textbox', { name: 'Handle' }), 'a/b')
    await user.click(screen.getByRole('button', { name: 'Save handle' }))

    expect(screen.getByRole('alert')).toHaveTextContent('3–30 characters')
    expect(updateMe).not.toHaveBeenCalled()
  })

  it('says when a handle is taken', async () => {
    vi.mocked(updateMe).mockRejectedValue(new ApiError(409, 'handle_taken', 'that handle is already taken'))
    const user = userEvent.setup()
    renderSharing()

    await user.type(await screen.findByRole('textbox', { name: 'Handle' }), 'taken')
    await user.click(screen.getByRole('button', { name: 'Save handle' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('That handle is taken. Try another.')
  })

  it('asks before going public, and does nothing if the user says no', async () => {
    vi.mocked(getMe).mockResolvedValue({ ...me, handle: 'rishekesh' })
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const user = userEvent.setup()
    renderSharing()

    await user.click(await screen.findByRole('checkbox', { name: 'Share this journal publicly' }))

    expect(confirm).toHaveBeenCalledWith(expect.stringContaining('every entry'))
    expect(updateJournal).not.toHaveBeenCalled()
    expect(screen.getByRole('checkbox', { name: 'Share this journal publicly' })).not.toBeChecked()
  })

  it('goes public after a yes, and shows the link to share', async () => {
    vi.mocked(getMe).mockResolvedValue({ ...me, handle: 'rishekesh' })
    vi.mocked(updateJournal).mockResolvedValue({ ...journal, visibility: 'public' })
    vi.spyOn(window, 'confirm').mockReturnValue(true)
    const user = userEvent.setup()
    renderSharing()

    await user.click(await screen.findByRole('checkbox', { name: 'Share this journal publicly' }))

    expect(updateJournal).toHaveBeenCalledWith({ visibility: 'public' })
    expect(await screen.findByText('visibility: public')).toBeInTheDocument()
    const link = screen.getByRole('link', { name: /\/j\/rishekesh$/ })
    expect(link).toHaveAttribute('href', '/j/rishekesh')
    expect(link).toHaveAttribute('rel', 'noopener noreferrer')
  })

  it('goes private without asking', async () => {
    vi.mocked(getMe).mockResolvedValue({ ...me, handle: 'rishekesh' })
    vi.mocked(updateJournal).mockResolvedValue(journal)
    const confirm = vi.spyOn(window, 'confirm')
    const user = userEvent.setup()
    renderSharing({ ...journal, visibility: 'public' })

    await user.click(await screen.findByRole('checkbox', { name: 'Share this journal publicly' }))

    expect(confirm).not.toHaveBeenCalled()
    expect(updateJournal).toHaveBeenCalledWith({ visibility: 'private' })
    expect(await screen.findByText('visibility: private')).toBeInTheDocument()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })

  it('warns that changing the handle of a public journal frees the old one', async () => {
    vi.mocked(getMe).mockResolvedValue({ ...me, handle: 'rishekesh' })
    const confirm = vi.spyOn(window, 'confirm').mockReturnValue(false)
    const user = userEvent.setup()
    renderSharing({ ...journal, visibility: 'public' })

    const input = await screen.findByRole('textbox', { name: 'Handle' })
    await user.clear(input)
    await user.type(input, 'rishi2')
    await user.click(screen.getByRole('button', { name: 'Save handle' }))

    expect(confirm).toHaveBeenCalledWith(expect.stringContaining('someone else could take “rishekesh”'))
    expect(updateMe).not.toHaveBeenCalled()
  })
})