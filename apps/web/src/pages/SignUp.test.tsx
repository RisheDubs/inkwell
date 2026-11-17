import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { registerUser } from '../lib/auth'
import SignUp from './SignUp'

// Fake only registerUser; keep the real authErrorMessage.
vi.mock('../lib/auth', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/auth')>()),
  registerUser: vi.fn(),
}))

const cognitoError = (name: string, message = name) => Object.assign(new Error(message), { name })

function ConfirmStub() {
  const state = useLocation().state as { email?: string } | null
  return <p>confirm page for {state?.email}</p>
}

function renderSignUp() {
  const router = createMemoryRouter(
    [
      { path: '/signup', element: <SignUp /> },
      { path: '/confirm', element: <ConfirmStub /> },
      { path: '/signin', element: <p>sign in page</p> },
    ],
    { initialEntries: ['/signup'] },
  )
  render(<RouterProvider router={router} />)
}

async function fillAndSubmit() {
  const user = userEvent.setup()
  await user.type(screen.getByLabelText('Email'), 'ana@example.com')
  await user.type(screen.getByLabelText('Password'), 'Str0ng!pass')
  await user.click(screen.getByRole('button', { name: 'Create account' }))
}

describe('SignUp', () => {
  // Braces matter: without them this would RETURN the mock, and Vitest runs a function
  // returned from beforeEach as cleanup after the test, calling registerUser() again.
  beforeEach(() => {
    vi.mocked(registerUser).mockReset()
  })

  it('registers the user and moves to the confirm page with the email in router state', async () => {
    vi.mocked(registerUser).mockResolvedValue(undefined)
    renderSignUp()
    await fillAndSubmit()

    expect(registerUser).toHaveBeenCalledWith('ana@example.com', 'Str0ng!pass')
    expect(await screen.findByText('confirm page for ana@example.com')).toBeInTheDocument()
  })

  it('shows a friendly error and stays put when Cognito rejects the sign-up', async () => {
    vi.mocked(registerUser).mockImplementation(async () => {
      throw cognitoError('UsernameExistsException', 'User already exists')
    })
    renderSignUp()
    await fillAndSubmit()

    expect(await screen.findByRole('alert')).toHaveTextContent('An account with this email already exists')
    expect(screen.getByRole('heading', { name: 'Create your account' })).toBeInTheDocument()
  })
})