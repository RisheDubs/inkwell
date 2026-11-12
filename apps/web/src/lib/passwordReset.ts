import { confirmResetPassword, resetPassword } from 'aws-amplify/auth'

// Step 1: ask Cognito to email a reset code. Never reveals whether the account exists.
export async function requestPasswordReset(email: string): Promise<void> {
  try {
    await resetPassword({ username: email })
  } catch (err) {
    // The pool already hides unknown emails (step 10); this is a second lock on the same door.
    if (err instanceof Error && err.name === 'UserNotFoundException') return
    throw err
  }
}

// Step 2: the emailed code plus the new password. Cognito checks the password policy.
export async function completePasswordReset(email: string, code: string, newPassword: string): Promise<void> {
  await confirmResetPassword({ username: email, confirmationCode: code.trim(), newPassword })
}