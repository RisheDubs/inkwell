import {
  confirmSignUp,
  fetchAuthSession,
  resendSignUpCode,
  signIn,
  signOut,
  signUp,
} from 'aws-amplify/auth'

// Thin wrappers over Amplify Auth. Pages call these, never Amplify directly (ADR-0007).

export async function registerUser(email: string, password: string): Promise<void> {
  await signUp({
    username: email, // the pool signs in with email
    password,
    options: { userAttributes: { email } }, // Cognito sends the code to this address
  })
}

export async function confirmRegistration(email: string, code: string): Promise<void> {
  await confirmSignUp({ username: email, confirmationCode: code.trim() })
}

export async function resendConfirmationCode(email: string): Promise<void> {
  await resendSignUpCode({ username: email })
}

export type SignInResult = 'signed-in' | 'needs-confirmation'

// SRP sign-in: the password never leaves the browser (step 10).
export async function signInUser(email: string, password: string): Promise<SignInResult> {
  const { isSignedIn: done, nextStep } = await signIn({ username: email, password })
  if (done) return 'signed-in'
  if (nextStep.signInStep === 'CONFIRM_SIGN_UP') return 'needs-confirmation'
  // MFA and other challenges aren't enabled on the pool; fail loudly rather than hang.
  throw Object.assign(new Error(`Unsupported sign-in step: ${nextStep.signInStep}`), {
    name: 'UnsupportedSignInStep',
  })
}

export async function signOutUser(): Promise<void> {
  await signOut()
}

// True when Amplify holds a working session. fetchAuthSession refreshes an expired access
// token with the refresh token first. Never throws: any failure means "not signed in".
export async function isSignedIn(): Promise<boolean> {
  try {
    const { tokens } = await fetchAuthSession()
    return tokens !== undefined
  } catch {
    return false
  }
}

// The token for API calls. ADR-0002: the API accepts ACCESS tokens only, never ID tokens.
export async function getAccessToken(): Promise<string | undefined> {
  const { tokens } = await fetchAuthSession()
  return tokens?.accessToken?.toString()
}

const MESSAGES: Record<string, string> = {
  UsernameExistsException: 'An account with this email already exists. Try signing in.',
  CodeMismatchException: 'That code is not right. Check the email and try again.',
  ExpiredCodeException: 'That code has expired. Send a new one.',
  LimitExceededException: 'Too many attempts. Wait a few minutes and try again.',
  TooManyRequestsException: 'Too many attempts. Wait a few minutes and try again.',
  NotAuthorizedException: 'Incorrect email or password.',
  UserNotConfirmedException: 'Please confirm your email first.',
  UserAlreadyAuthenticatedException: 'You are already signed in.',
  UnsupportedSignInStep: "This account needs a sign-in step Inkwell doesn't support yet.",
}

// Turns a Cognito error into a message that is safe and useful to show.
export function authErrorMessage(err: unknown): string {
  if (!(err instanceof Error)) return 'Something went wrong. Please try again.'
  if (err.name === 'InvalidPasswordException') {
    // Cognito's text names the password rule that failed, which helps the user.
    return err.message.replace(/^Password did not conform with policy: /, '')
  }
  return MESSAGES[err.name] ?? 'Something went wrong. Please try again.'
}