import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { isSignedIn } from './lib/auth'
import { listEntries } from './lib/entries'
import { routes } from './router'

vi.mock('./lib/auth', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./lib/auth')>()),
  isSignedIn: vi.fn(),
}))
vi.mock('./lib/entries', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./lib/entries')>()),
  listEntries: vi.fn(),
}))

// A memory router keeps the URL in memory, so tests can start on any path without a browser.
function renderAt(path: string) {
  render(<RouterProvider router={createMemoryRouter(routes, { initialEntries: [path] })} />)
}

describe('routes', () => {
  beforeEach(() => {
    vi.mocked(isSignedIn).mockResolvedValue(true)
    vi.mocked(listEntries).mockResolvedValue({ entries: [] })
  })

  it.each([
    ['/', 'Your journal'],
    ['/entries/new', 'New entry'],
    ['/signin', 'Sign in'],
    ['/signup', 'Create your account'],
    ['/confirm', 'Confirm your email'],
    ['/reset-password', 'Reset your password'],
    ['/no-such-page', 'Page not found'],
  ])('%s shows "%s"', async (path, heading) => {
    renderAt(path)
    // findBy waits: the guard on "/" answers asynchronously.
    expect(await screen.findByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
  })

  it('sends a signed-out visitor from the journal to sign-in', async () => {
    vi.mocked(isSignedIn).mockResolvedValue(false)
    renderAt('/')
    expect(await screen.findByRole('heading', { level: 1, name: 'Sign in' })).toBeInTheDocument()
  })
})