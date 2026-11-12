import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ApiError, getMe } from '../lib/api'
import Home from './Home'

// Fake getMe; keep the real ApiError class so instanceof checks work.
vi.mock('../lib/api', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/api')>()),
  getMe: vi.fn(),
}))

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

describe('Home', () => {
  beforeEach(() => {
    vi.mocked(getMe).mockReset()
  })

  it('shows who is signed in, from the API', async () => {
    vi.mocked(getMe).mockResolvedValue({ id: 'u1', email: 'ana@example.com' })
    renderHome()

    expect(screen.getByRole('status')).toHaveTextContent('Loading your account')
    expect(await screen.findByText('Signed in as ana@example.com')).toBeInTheDocument()
  })

  it('sends the user to sign-in when the API rejects the token', async () => {
    vi.mocked(getMe).mockImplementation(async () => {
      throw new ApiError(401, 'unauthorized', 'invalid or expired access token')
    })
    renderHome()

    expect(await screen.findByText('sign in page, came from /')).toBeInTheDocument()
  })

  it('shows a friendly error with the request ID for other failures', async () => {
    vi.mocked(getMe).mockImplementation(async () => {
      throw new ApiError(500, 'internal', 'boom', 'req-123')
    })
    renderHome()

    expect(await screen.findByRole('alert')).toHaveTextContent(
      "We couldn't load your account. Please try again. (reference: req-123)",
    )
  })
})