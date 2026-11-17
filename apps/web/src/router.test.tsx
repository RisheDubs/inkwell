import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { isSignedIn } from './lib/auth'
import { listEntries } from './lib/entries'
import { getJournal } from './lib/journal'
import { routes } from './router'

vi.mock('./lib/auth', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./lib/auth')>()),
  isSignedIn: vi.fn(),
}))
vi.mock('./lib/entries', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./lib/entries')>()),
  listEntries: vi.fn(),
}))
vi.mock('./lib/journal', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./lib/journal')>()),
  getJournal: vi.fn(),
}))

// A memory router keeps the URL in memory, so tests can start on any path without a browser.
function renderAt(path: string) {
  render(<RouterProvider router={createMemoryRouter(routes, { initialEntries: [path] })} />)
}

describe('routes', () => {
  beforeEach(() => {
    vi.mocked(isSignedIn).mockResolvedValue(true)
    vi.mocked(listEntries).mockResolvedValue({ entries: [] })
    vi.mocked(getJournal).mockResolvedValue({
      id: 'j1',
      title: 'My Journal',
      visibility: 'private',
      theme: { paper: 'dotted', font: 'serif', accent: '#2f5d50' },
      createdAt: '2026-11-03T09:00:00Z',
    })
  })

  it.each([
    ['/', 'Your journal'],
    ['/entries/new', 'New entry'],
    ['/settings', 'Journal settings'],
    ['/signin', 'Sign in'],
    ['/signup', 'Create your account'],
    ['/confirm', 'Confirm your email'],
    ['/reset-password', 'Reset your password'],
    ['/no-such-page', 'Page not found'],
  ])('%s shows "%s"', async (path, heading) => {
    renderAt(path)
    // findBy waits: the guard and the journal load asynchronously.
    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
  })

  it('sends a signed-out visitor from the journal to sign-in', async () => {
    vi.mocked(isSignedIn).mockResolvedValue(false)
    renderAt('/')
    expect(await screen.findByRole('heading', { level: 1, name: 'Sign in' })).toBeInTheDocument()
  })
})