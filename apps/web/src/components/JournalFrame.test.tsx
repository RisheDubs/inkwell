import { render, screen, waitFor } from '@testing-library/react'
import { createMemoryRouter, RouterProvider, useOutletContext } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getJournal, type Journal } from '../lib/journal'
import JournalFrame, { type JournalContext } from './JournalFrame'

vi.mock('../lib/journal', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/journal')>()),
  getJournal: vi.fn(),
}))

const journal: Journal = {
  id: 'j1',
  title: 'Rishi’s Journal',
  visibility: 'private',
  theme: { paper: 'grid', font: 'mono', accent: '#aa3366' },
  createdAt: '2026-11-03T09:00:00Z',
}

function TitleProbe() {
  const { journal: current } = useOutletContext<JournalContext>()
  return <p>journal: {current?.title ?? 'loading'}</p>
}

function renderFrame() {
  const router = createMemoryRouter([{ element: <JournalFrame />, children: [{ path: '/', element: <TitleProbe /> }] }])
  render(<RouterProvider router={router} />)
}

describe('JournalFrame', () => {
  beforeEach(() => {
    vi.mocked(getJournal).mockReset()
  })

  it('applies the journal’s theme and shares the journal with the page inside', async () => {
    vi.mocked(getJournal).mockResolvedValue(journal)
    renderFrame()

    expect(await screen.findByText('journal: Rishi’s Journal')).toBeInTheDocument()
    const frame = screen.getByTestId('journal-frame')
    expect(frame).toHaveClass('journal', 'paper-grid')
    expect(frame.style.getPropertyValue('--ink-accent')).toBe('#aa3366')
  })

  it('keeps the default look if the journal cannot be loaded', async () => {
    vi.mocked(getJournal).mockRejectedValue(new Error('network down'))
    renderFrame()

    await waitFor(() => expect(getJournal).toHaveBeenCalled())
    expect(screen.getByTestId('journal-frame')).toHaveClass('paper-dotted')
    expect(screen.getByText('journal: loading')).toBeInTheDocument()
  })
})