import { useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { authErrorMessage, confirmRegistration, resendConfirmationCode } from '../lib/auth'

const inputClass = 'mt-1 w-full rounded border border-stone-300 p-2'

export default function ConfirmSignUp() {
  const navigate = useNavigate()
  const state = useLocation().state as { email?: string } | null
  // Prefilled from sign-up, but editable, so a user who arrives later can still confirm.
  const [email, setEmail] = useState(state?.email ?? '')
  const [code, setCode] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const [busy, setBusy] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setNotice('')
    setBusy(true)
    try {
      await confirmRegistration(email.trim(), code)
      navigate('/signin', { state: { email: email.trim(), confirmed: true } })
    } catch (err) {
      setError(authErrorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  async function handleResend() {
    setError('')
    setNotice('')
    try {
      await resendConfirmationCode(email.trim())
      setNotice('We sent a new code. Check your email.')
    } catch (err) {
      setError(authErrorMessage(err))
    }
  }

  return (
    <div className="max-w-sm">
      <h1 className="text-2xl font-semibold">Confirm your email</h1>
      <p className="mt-1 text-sm text-stone-600">Enter the 6-digit code we emailed you.</p>
      <form onSubmit={handleSubmit} className="mt-4 space-y-3">
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
        {error && (
          <p role="alert" className="text-sm text-red-700">
            {error}
          </p>
        )}
        {notice && (
          <p role="status" className="text-sm text-green-700">
            {notice}
          </p>
        )}
        <div className="flex items-center gap-4">
          <button
            type="submit"
            disabled={busy}
            className="rounded bg-stone-900 px-4 py-2 text-white disabled:opacity-50"
          >
            {busy ? 'Confirming…' : 'Confirm'}
          </button>
          <button type="button" onClick={handleResend} disabled={!email.trim()} className="text-sm underline">
            Send a new code
          </button>
        </div>
      </form>
    </div>
  )
}