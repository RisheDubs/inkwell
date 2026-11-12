import { confirmSignUp, fetchAuthSession, signIn, signOut, signUp } from 'aws-amplify/auth'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import {
  authErrorMessage,
  confirmRegistration,
  getAccessToken,
  isSignedIn,
  registerUser,
  signInUser,
  signOutUser,
} from './auth'

// Replace Amplify with fakes: unit tests must never call real Cognito.
vi.mock('aws-amplify/auth', () => ({
  signUp: vi.fn(),
  confirmSignUp: vi.fn(),
  resendSignUpCode: vi.fn(),
  signIn: vi.fn(),
  signOut: vi.fn(),
  fetchAuthSession: vi.fn(),
}))

const cognitoError = (name: string, message = name) => Object.assign(new Error(message), { name })

// Builds a fake Amplify session holding the given tokens.
type Session = Awaited<ReturnType<typeof fetchAuthSession>>
const sessionWith = (tokens: unknown) => ({ tokens }) as Session
const token = (jwt: string) => ({ toString: () => jwt })

beforeEach(() => {
  vi.resetAllMocks() // braces: beforeEach must not return anything (see 11b)
})

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

describe('signInUser', () => {
  it('reports signed-in when Cognito is done', async () => {
    vi.mocked(signIn).mockResolvedValue({ isSignedIn: true, nextStep: { signInStep: 'DONE' } })
    await expect(signInUser('ana@example.com', 'pw')).resolves.toBe('signed-in')
    expect(signIn).toHaveBeenCalledWith({ username: 'ana@example.com', password: 'pw' })
  })

  it('reports needs-confirmation for a user who never entered the email code', async () => {
    vi.mocked(signIn).mockResolvedValue({ isSignedIn: false, nextStep: { signInStep: 'CONFIRM_SIGN_UP' } })
    await expect(signInUser('ana@example.com', 'pw')).resolves.toBe('needs-confirmation')
  })

  it('fails loudly on a step we do not support, such as MFA', async () => {
    vi.mocked(signIn).mockResolvedValue({
      isSignedIn: false,
      nextStep: { signInStep: 'CONFIRM_SIGN_IN_WITH_TOTP_CODE' },
    } as Awaited<ReturnType<typeof signIn>>)
    await expect(signInUser('ana@example.com', 'pw')).rejects.toMatchObject({ name: 'UnsupportedSignInStep' })
  })
})

describe('signOutUser', () => {
  it('signs out of Amplify', async () => {
    await signOutUser()
    expect(signOut).toHaveBeenCalledOnce()
  })
})

describe('isSignedIn', () => {
  it('is true when the session holds tokens', async () => {
    vi.mocked(fetchAuthSession).mockResolvedValue(sessionWith({ accessToken: token('a') }))
    await expect(isSignedIn()).resolves.toBe(true)
  })

  it('is false when there is no session', async () => {
    vi.mocked(fetchAuthSession).mockResolvedValue(sessionWith(undefined))
    await expect(isSignedIn()).resolves.toBe(false)
  })

  it('is false, not an error, when the session check fails', async () => {
    vi.mocked(fetchAuthSession).mockImplementation(async () => {
      throw cognitoError('NetworkError')
    })
    await expect(isSignedIn()).resolves.toBe(false)
  })
})

describe('getAccessToken', () => {
  it('returns the access token, never the ID token (ADR-0002)', async () => {
    vi.mocked(fetchAuthSession).mockResolvedValue(
      sessionWith({ accessToken: token('access-jwt'), idToken: token('id-jwt') }),
    )
    await expect(getAccessToken()).resolves.toBe('access-jwt')
  })

  it('returns undefined when signed out', async () => {
    vi.mocked(fetchAuthSession).mockResolvedValue(sessionWith(undefined))
    await expect(getAccessToken()).resolves.toBeUndefined()
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