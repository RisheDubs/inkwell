import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { isSignedIn } from '../lib/auth'
import RequireAuth from './RequireAuth'

vi.mock('../lib/auth', () => ({ isSignedIn: vi.fn() }))

function SignInStub() {
  const state = useLocation().state as { from?: string } | null
  return <p>sign in page, came from {state?.from ?? 'nowhere'}</p>
}

// The guard wraps a protected route, like it will in router.tsx.
function renderAt(path: string) {
  const router = createMemoryRouter(
    [
      { element: <RequireAuth />, children: [{ path: '/entries/:id', element: <p>secret page</p> }] },
      { path: '/signin', element: <SignInStub /> },
    ],
    { initialEntries: [path] },
  )
  render(<RouterProvider router={router} />)
}

describe('RequireAuth', () => {
  beforeEach(() => {
    vi.mocked(isSignedIn).mockReset()
  })

  it('shows the protected page to a signed-in user', async () => {
    vi.mocked(isSignedIn).mockResolvedValue(true)
    renderAt('/entries/42')
    expect(await screen.findByText('secret page')).toBeInTheDocument()
  })

  it('sends a signed-out user to sign-in, remembering where they were going', async () => {
    vi.mocked(isSignedIn).mockResolvedValue(false)
    renderAt('/entries/42')
    expect(await screen.findByText('sign in page, came from /entries/42')).toBeInTheDocument()
    expect(screen.queryByText('secret page')).not.toBeInTheDocument()
  })

  it('shows a status, and nothing protected, while the session check is running', () => {
    vi.mocked(isSignedIn).mockImplementation(() => new Promise(() => {})) // never settles
    renderAt('/entries/42')
    expect(screen.getByRole('status')).toHaveTextContent('Checking your session')
    expect(screen.queryByText('secret page')).not.toBeInTheDocument()
  })
})