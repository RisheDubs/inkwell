import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { signInUser } from '../lib/auth'
import SignIn from './SignIn'

vi.mock('../lib/auth', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/auth')>()),
  signInUser: vi.fn(),
}))

const cognitoError = (name: string, message = name) => Object.assign(new Error(message), { name })

function ConfirmStub() {
  const state = useLocation().state as { email?: string } | null
  return <p>confirm page for {state?.email}</p>
}

// Open /signin with whatever router state the guard or sign-up would have passed.
function renderSignIn(state?: object) {
  const router = createMemoryRouter(
    [
      { path: '/signin', element: <SignIn /> },
      { path: '/', element: <p>home page</p> },
      { path: '/entries/:id', element: <p>entry page</p> },
      { path: '/confirm', element: <ConfirmStub /> },
    ],
    { initialEntries: [{ pathname: '/signin', state }] },
  )
  render(<RouterProvider router={router} />)
}

async function signIn({ typeEmail = true } = {}) {
  const user = userEvent.setup()
  if (typeEmail) await user.type(screen.getByLabelText('Email'), 'ana@example.com')
  await user.type(screen.getByLabelText('Password'), 'Str0ng!pass')
  await user.click(screen.getByRole('button', { name: 'Sign in' }))
}

describe('SignIn', () => {
  beforeEach(() => {
    vi.mocked(signInUser).mockReset()
  })

  it('signs in and returns to the page the guard sent them from', async () => {
    vi.mocked(signInUser).mockResolvedValue('signed-in')
    renderSignIn({ from: '/entries/42' })
    await signIn()

    expect(signInUser).toHaveBeenCalledWith('ana@example.com', 'Str0ng!pass')
    expect(await screen.findByText('entry page')).toBeInTheDocument()
  })

  it('goes to the journal home when there is no return path', async () => {
    vi.mocked(signInUser).mockResolvedValue('signed-in')
    renderSignIn()
    await signIn()

    expect(await screen.findByText('home page')).toBeInTheDocument()
  })

  it('never redirects off-site, even if the return path says so', async () => {
    vi.mocked(signInUser).mockResolvedValue('signed-in')
    renderSignIn({ from: '//evil.example' })
    await signIn()

    expect(await screen.findByText('home page')).toBeInTheDocument()
  })

  it('sends a user who never confirmed their email to the confirm page', async () => {
    vi.mocked(signInUser).mockResolvedValue('needs-confirmation')
    renderSignIn()
    await signIn()

    expect(await screen.findByText('confirm page for ana@example.com')).toBeInTheDocument()
  })

  it('shows the same message for a wrong password as for an unknown email', async () => {
    vi.mocked(signInUser).mockImplementation(async () => {
      throw cognitoError('NotAuthorizedException', 'Incorrect username or password.')
    })
    renderSignIn()
    await signIn()

    expect(await screen.findByRole('alert')).toHaveTextContent('Incorrect email or password.')
  })

  it('prefills the email and says it is confirmed when coming from sign-up', () => {
    renderSignIn({ email: 'ana@example.com', confirmed: true })

    expect(screen.getByLabelText('Email')).toHaveValue('ana@example.com')
    expect(screen.getByRole('status')).toHaveTextContent('Email confirmed')
  })
})