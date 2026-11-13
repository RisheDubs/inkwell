import { fireEvent, render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { createMemoryRouter, Outlet, RouterProvider } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import type { JournalContext } from '../components/JournalFrame'
import { ApiError } from '../lib/api'
import { updateJournal, type Journal } from '../lib/journal'
import Settings from './Settings'

vi.mock('../lib/journal', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/journal')>()),
  updateJournal: vi.fn(),
}))

const journal: Journal = {
  id: 'j1',
  title: 'Rishi’s Journal',
  visibility: 'private',
  theme: { paper: 'dotted', font: 'serif', accent: '#2f5d50' },
  createdAt: '2026-11-03T09:00:00Z',
}

// Stands in for JournalFrame: holds the journal and shows what the page saved into it.
function FrameStub({ initial }: { initial: Journal | null }) {
  const [current, setCurrent] = useState(initial)
  return (
    <>
      <p>frame font: {current?.theme.font ?? 'none'}</p>
      <Outlet context={{ journal: current, setJournal: setCurrent } satisfies JournalContext} />
    </>
  )
}

function renderSettings(initial: Journal | null = journal) {
  const router = createMemoryRouter(
    [{ element: <FrameStub initial={initial} />, children: [{ path: '/settings', element: <Settings /> }] }],
    { initialEntries: ['/settings'] },
  )
  render(<RouterProvider router={router} />)
}

describe('Settings', () => {
  beforeEach(() => {
    vi.mocked(updateJournal).mockReset()
  })

  it('waits for the journal before showing the form', () => {
    renderSettings(null)
    expect(screen.getByRole('status')).toHaveTextContent('Loading your journal')
  })

  it('fills the form from the journal', () => {
    renderSettings()
    expect(screen.getByRole('textbox', { name: 'Journal title' })).toHaveValue('Rishi’s Journal')
    expect(screen.getByRole('radio', { name: 'dotted' })).toBeChecked()
    expect(screen.getByLabelText('Font')).toHaveValue('serif')
    expect(screen.getByLabelText('Accent colour')).toHaveValue('#2f5d50')
  })

  it('saves only what changed and restyles the journal straight away', async () => {
    vi.mocked(updateJournal).mockResolvedValue({ ...journal, theme: { ...journal.theme, font: 'sans', accent: '#aa3366' } })
    const user = userEvent.setup()
    renderSettings()

    await user.selectOptions(screen.getByLabelText('Font'), 'sans')
    fireEvent.change(screen.getByLabelText('Accent colour'), { target: { value: '#aa3366' } })
    await user.click(screen.getByRole('button', { name: 'Save settings' }))

    expect(updateJournal).toHaveBeenCalledWith({ theme: { font: 'sans', accent: '#aa3366' } })
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
    expect(screen.getByText('frame font: sans')).toBeInTheDocument()
  })

  it('does not call the API when nothing changed', async () => {
    const user = userEvent.setup()
    renderSettings()

    await user.click(screen.getByRole('button', { name: 'Save settings' }))

    expect(updateJournal).not.toHaveBeenCalled()
    expect(screen.getByText('Nothing to save.')).toBeInTheDocument()
  })

  it("shows the API's message when it rejects a value", async () => {
    vi.mocked(updateJournal).mockImplementation(async () => {
      throw new ApiError(400, 'invalid_request', 'title: must be at most 100 characters')
    })
    const user = userEvent.setup()
    renderSettings()

    await user.type(screen.getByRole('textbox', { name: 'Journal title' }), '!')
    await user.click(screen.getByRole('button', { name: 'Save settings' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('title: must be at most 100 characters')
  })
})