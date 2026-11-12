import { confirmSignUp, resendSignUpCode, signUp } from 'aws-amplify/auth'

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

const MESSAGES: Record<string, string> = {
  UsernameExistsException: 'An account with this email already exists. Try signing in.',
  CodeMismatchException: 'That code is not right. Check the email and try again.',
  ExpiredCodeException: 'That code has expired. Send a new one.',
  LimitExceededException: 'Too many attempts. Wait a few minutes and try again.',
  TooManyRequestsException: 'Too many attempts. Wait a few minutes and try again.',
  NotAuthorizedException: 'Incorrect email or password.',
  UserNotConfirmedException: 'Please confirm your email first.',
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