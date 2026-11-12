import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { isSignedIn, signOutUser } from '../lib/auth'
import Layout from './Layout'

vi.mock('../lib/auth', () => ({ isSignedIn: vi.fn(), signOutUser: vi.fn() }))

function renderLayout() {
  const router = createMemoryRouter(
    [
      {
        element: <Layout />,
        children: [
          { path: '/', element: <p>home page</p> },
          { path: '/signin', element: <p>sign in page</p> },
        ],
      },
    ],
    { initialEntries: ['/'] },
  )
  render(<RouterProvider router={router} />)
}

describe('Layout', () => {
  beforeEach(() => {
    vi.mocked(isSignedIn).mockReset()
    vi.mocked(signOutUser).mockReset()
  })

  it('offers sign in and sign up to a signed-out visitor', async () => {
    vi.mocked(isSignedIn).mockResolvedValue(false)
    renderLayout()

    expect(await screen.findByRole('link', { name: 'Sign in' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sign up' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Sign out' })).not.toBeInTheDocument()
  })

  it('signs a signed-in user out and returns them to sign-in', async () => {
    vi.mocked(isSignedIn).mockResolvedValue(true)
    // Like the real thing: once signed out, the session check says "no".
    vi.mocked(signOutUser).mockImplementation(async () => {
      vi.mocked(isSignedIn).mockResolvedValue(false)
    })
    const user = userEvent.setup()
    renderLayout()

    await user.click(await screen.findByRole('button', { name: 'Sign out' }))

    expect(signOutUser).toHaveBeenCalledOnce()
    expect(await screen.findByText('sign in page')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Sign in' })).toBeInTheDocument()
  })
})