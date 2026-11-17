import { confirmResetPassword, resetPassword } from 'aws-amplify/auth'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { completePasswordReset, requestPasswordReset } from './passwordReset'

vi.mock('aws-amplify/auth', () => ({ resetPassword: vi.fn(), confirmResetPassword: vi.fn() }))

const cognitoError = (name: string) => Object.assign(new Error(name), { name })

beforeEach(() => {
  vi.resetAllMocks()
})

describe('requestPasswordReset', () => {
  it('asks Cognito to send a code to the email', async () => {
    await requestPasswordReset('ana@example.com')
    expect(resetPassword).toHaveBeenCalledWith({ username: 'ana@example.com' })
  })

  it('looks the same for an unknown email as for a real one', async () => {
    vi.mocked(resetPassword).mockImplementation(async () => {
      throw cognitoError('UserNotFoundException')
    })
    await expect(requestPasswordReset('nobody@example.com')).resolves.toBeUndefined()
  })

  it('still reports real problems, such as too many attempts', async () => {
    vi.mocked(resetPassword).mockImplementation(async () => {
      throw cognitoError('LimitExceededException')
    })
    await expect(requestPasswordReset('ana@example.com')).rejects.toMatchObject({ name: 'LimitExceededException' })
  })
})

describe('completePasswordReset', () => {
  it('sends the trimmed code with the new password', async () => {
    await completePasswordReset('ana@example.com', ' 123456 ', 'N3w!password')
    expect(confirmResetPassword).toHaveBeenCalledWith({
      username: 'ana@example.com',
      confirmationCode: '123456',
      newPassword: 'N3w!password',
    })
  })
})