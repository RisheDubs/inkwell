import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate } from 'react-router'
import { authErrorMessage, signInUser } from '../lib/auth'

const inputClass = 'mt-1 w-full rounded border border-stone-300 p-2'

type SignInState = { email?: string; confirmed?: boolean; from?: string } | null

// Only paths inside this app. "/entries/42" is fine; "//evil.example" would leave the site
// (an open redirect), so anything odd falls back to the journal home.
function safeReturnPath(from: unknown): string {
  return typeof from === 'string' && from.startsWith('/') && !from.startsWith('//') ? from : '/'
}

export default function SignIn() {
  const navigate = useNavigate()
  const state = useLocation().state as SignInState
  const [email, setEmail] = useState(state?.email ?? '')
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setBusy(true)
    try {
      const result = await signInUser(email.trim(), password)
      if (result === 'needs-confirmation') {
        navigate('/confirm', { state: { email: email.trim() } })
      } else {
        // replace: Back from the journal shouldn't land on the sign-in form again.
        navigate(safeReturnPath(state?.from), { replace: true })
      }
    } catch (err) {
      setError(authErrorMessage(err))
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="max-w-sm">
      <h1 className="text-2xl font-semibold">Sign in</h1>
      {state?.confirmed && (
        <p role="status" className="mt-1 text-sm text-green-700">
          Email confirmed. Sign in to continue.
        </p>
      )}
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
          <span className="text-sm">Password</span>
          <input
            type="password"
            required
            autoComplete="current-password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className={inputClass}
          />
        </label>
        {error && (
          <p role="alert" className="text-sm text-red-700">
            {error}
          </p>
        )}
        <button
          type="submit"
          disabled={busy}
          className="rounded bg-stone-900 px-4 py-2 text-white disabled:opacity-50"
        >
          {busy ? 'Signing in…' : 'Sign in'}
        </button>
      </form>
      <p className="mt-4 text-sm">
        <Link to="/reset-password" className="underline">Forgot your password?</Link>
      </p>
      <p className="mt-2 text-sm">
        New here? <Link to="/signup" className="underline">Create an account</Link>
      </p>
    </div>
  )
}