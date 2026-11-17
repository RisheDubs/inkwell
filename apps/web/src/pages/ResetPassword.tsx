import { useState, type FormEvent } from 'react'
import { useNavigate } from 'react-router'
import { authErrorMessage } from '../lib/auth'
import { completePasswordReset, requestPasswordReset } from '../lib/passwordReset'

const inputClass = 'mt-1 w-full rounded border border-stone-300 p-2'

export default function ResetPassword() {
  const navigate = useNavigate()
  const [stage, setStage] = useState<'request' | 'confirm'>('request')
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  // Shared wrapper: clear the old error, disable the button, show a friendly error on failure.
  async function run(action: () => Promise<void>) {
    setError('')
    setBusy(true)
    try {
      await action()
    } catch (err) {
      setError(authErrorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  function handleRequest(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    void run(async () => {
      await requestPasswordReset(email.trim())
      setStage('confirm')
    })
  }

  function handleConfirm(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    void run(async () => {
      await completePasswordReset(email.trim(), code, newPassword)
      navigate('/signin', { replace: true, state: { email: email.trim() } })
    })
  }

  const errorMessage = error && (
    <p role="alert" className="text-sm text-red-700">
      {error}
    </p>
  )

  return (
    <div className="max-w-sm">
      <h1 className="text-2xl font-semibold">Reset your password</h1>

      {stage === 'request' ? (
        <form onSubmit={handleRequest} className="mt-4 space-y-3">
          <p className="text-sm text-stone-600">Enter your email and we'll send you a code.</p>
          <label className="block">
            <span className="text-sm">Email</span>
            <input
              type="email"
              required
              autoComplete="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              className={inputClass}
            />
          </label>
          {errorMessage}
          <button
            type="submit"
            disabled={busy}
            className="rounded bg-stone-900 px-4 py-2 text-white disabled:opacity-50"
          >
            {busy ? 'Sending…' : 'Send code'}
          </button>
        </form>
      ) : (
        <form onSubmit={handleConfirm} className="mt-4 space-y-3">
          {/* Same words whether or not the account exists: no user enumeration. */}
          <p role="status" className="text-sm text-stone-600">
            If an account exists for {email.trim()}, we sent a code to it. Check your email.
          </p>
          <label className="block">
            <span className="text-sm">Code</span>
            <input
              required
              inputMode="numeric"
              autoComplete="one-time-code"
              pattern="[0-9]{6}"
              value={code}
              onChange={(e) => setCode(e.target.value)}
              className={inputClass}
            />
          </label>
          <label className="block">
            <span className="text-sm">New password</span>
            <input
              type="password"
              required
              minLength={8}
              autoComplete="new-password"
              value={newPassword}
              onChange={(e) => setNewPassword(e.target.value)}
              className={inputClass}
            />
          </label>
          {errorMessage}
          <div className="flex items-center gap-4">
            <button
              type="submit"
              disabled={busy}
              className="rounded bg-stone-900 px-4 py-2 text-white disabled:opacity-50"
            >
              {busy ? 'Saving…' : 'Set new password'}
            </button>
            <button type="button" onClick={() => setStage('request')} className="text-sm underline">
              Use a different email
            </button>
          </div>
        </form>
      )}
    </div>
  )
}