import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider, useLocation } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { completePasswordReset, requestPasswordReset } from '../lib/passwordReset'
import ResetPassword from './ResetPassword'

vi.mock('../lib/passwordReset', () => ({ requestPasswordReset: vi.fn(), completePasswordReset: vi.fn() }))

const cognitoError = (name: string) => Object.assign(new Error(name), { name })

function SignInStub() {
  const state = useLocation().state as { email?: string } | null
  return <p>sign in page for {state?.email}</p>
}

function renderReset() {
  const router = createMemoryRouter(
    [
      { path: '/reset-password', element: <ResetPassword /> },
      { path: '/signin', element: <SignInStub /> },
    ],
    { initialEntries: ['/reset-password'] },
  )
  render(<RouterProvider router={router} />)
}

async function requestCode(user: ReturnType<typeof userEvent.setup>) {
  await user.type(screen.getByLabelText('Email'), 'ana@example.com')
  await user.click(screen.getByRole('button', { name: 'Send code' }))
}

describe('ResetPassword', () => {
  beforeEach(() => {
    vi.mocked(requestPasswordReset).mockReset()
    vi.mocked(completePasswordReset).mockReset()
  })

  it('asks for a code without saying whether the account exists', async () => {
    vi.mocked(requestPasswordReset).mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderReset()
    await requestCode(user)

    expect(requestPasswordReset).toHaveBeenCalledWith('ana@example.com')
    expect(await screen.findByRole('status')).toHaveTextContent('If an account exists for ana@example.com')
  })

  it('sets the new password and goes to sign-in with the email filled in', async () => {
    vi.mocked(requestPasswordReset).mockResolvedValue(undefined)
    vi.mocked(completePasswordReset).mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderReset()
    await requestCode(user)

    await user.type(await screen.findByLabelText('Code'), '123456')
    await user.type(screen.getByLabelText('New password'), 'N3w!password')
    await user.click(screen.getByRole('button', { name: 'Set new password' }))

    expect(completePasswordReset).toHaveBeenCalledWith('ana@example.com', '123456', 'N3w!password')
    expect(await screen.findByText('sign in page for ana@example.com')).toBeInTheDocument()
  })

  it('shows a friendly error for a wrong code', async () => {
    vi.mocked(requestPasswordReset).mockResolvedValue(undefined)
    vi.mocked(completePasswordReset).mockImplementation(async () => {
      throw cognitoError('CodeMismatchException')
    })
    const user = userEvent.setup()
    renderReset()
    await requestCode(user)

    await user.type(await screen.findByLabelText('Code'), '000000')
    await user.type(screen.getByLabelText('New password'), 'N3w!password')
    await user.click(screen.getByRole('button', { name: 'Set new password' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('That code is not right')
  })
})