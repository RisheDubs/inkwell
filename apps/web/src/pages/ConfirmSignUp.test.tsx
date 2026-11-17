import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { confirmRegistration, resendConfirmationCode } from '../lib/auth'
import ConfirmSignUp from './ConfirmSignUp'

vi.mock('../lib/auth', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/auth')>()),
  confirmRegistration: vi.fn(),
  resendConfirmationCode: vi.fn(),
}))

const cognitoError = (name: string, message = name) => Object.assign(new Error(message), { name })

// Start on /confirm as if we just came from sign-up, with the email in router state.
function renderConfirm() {
  const router = createMemoryRouter(
    [
      { path: '/confirm', element: <ConfirmSignUp /> },
      { path: '/signin', element: <p>sign in page</p> },
    ],
    { initialEntries: [{ pathname: '/confirm', state: { email: 'ana@example.com' } }] },
  )
  render(<RouterProvider router={router} />)
}

describe('ConfirmSignUp', () => {
  beforeEach(() => {
    vi.mocked(confirmRegistration).mockReset()
    vi.mocked(resendConfirmationCode).mockReset()
  })

  it('prefills the email, confirms the code and goes to sign-in', async () => {
    vi.mocked(confirmRegistration).mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderConfirm()

    expect(screen.getByLabelText('Email')).toHaveValue('ana@example.com')
    await user.type(screen.getByLabelText('Code'), '123456')
    await user.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(confirmRegistration).toHaveBeenCalledWith('ana@example.com', '123456')
    expect(await screen.findByText('sign in page')).toBeInTheDocument()
  })

  it('shows a friendly error for a wrong code', async () => {
    // Reject only when called, so the page's try/catch handles it straight away.
    vi.mocked(confirmRegistration).mockImplementation(async () => {
      throw cognitoError('CodeMismatchException', 'Invalid code')
    })
    const user = userEvent.setup()
    renderConfirm()

    await user.type(screen.getByLabelText('Code'), '000000')
    await user.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('That code is not right')
  })

  it('resends the code to the prefilled email', async () => {
    vi.mocked(resendConfirmationCode).mockResolvedValue(undefined)
    const user = userEvent.setup()
    renderConfirm()

    await user.click(screen.getByRole('button', { name: 'Send a new code' }))

    expect(resendConfirmationCode).toHaveBeenCalledWith('ana@example.com')
    expect(await screen.findByRole('status')).toHaveTextContent('We sent a new code')
  })
})