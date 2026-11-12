import { confirmSignUp, signUp } from 'aws-amplify/auth'
import { describe, expect, it, vi } from 'vitest'
import { authErrorMessage, confirmRegistration, registerUser } from './auth'

// Replace Amplify with fakes: unit tests must never call real Cognito.
vi.mock('aws-amplify/auth', () => ({
  signUp: vi.fn(),
  confirmSignUp: vi.fn(),
  resendSignUpCode: vi.fn(),
}))

const cognitoError = (name: string, message = name) => Object.assign(new Error(message), { name })

describe('registerUser', () => {
  it('uses the email as the username and as the email attribute', async () => {
    await registerUser('ana@example.com', 'Str0ng!pass')
    expect(signUp).toHaveBeenCalledWith({
      username: 'ana@example.com',
      password: 'Str0ng!pass',
      options: { userAttributes: { email: 'ana@example.com' } },
    })
  })
})

describe('confirmRegistration', () => {
  it('trims spaces pasted around the code', async () => {
    await confirmRegistration('ana@example.com', ' 123456 ')
    expect(confirmSignUp).toHaveBeenCalledWith({ username: 'ana@example.com', confirmationCode: '123456' })
  })
})

describe('authErrorMessage', () => {
  it('maps a known Cognito error to plain words', () => {
    expect(authErrorMessage(cognitoError('CodeMismatchException'))).toBe(
      'That code is not right. Check the email and try again.',
    )
  })

  it('keeps the failed password rule but drops the jargon', () => {
    const err = cognitoError(
      'InvalidPasswordException',
      'Password did not conform with policy: Password must have uppercase characters',
    )
    expect(authErrorMessage(err)).toBe('Password must have uppercase characters')
  })

  it('falls back to a generic message for anything unknown', () => {
    expect(authErrorMessage(cognitoError('SomethingNewException'))).toBe('Something went wrong. Please try again.')
    expect(authErrorMessage('not even an error')).toBe('Something went wrong. Please try again.')
  })
})