import { useEffect, useState } from 'react'
import { Navigate, Outlet, useLocation } from 'react-router'
import { isSignedIn } from '../lib/auth'

type Status = 'checking' | 'signed-in' | 'signed-out'

// Route guard: renders the protected child route only for a signed-in user.
// UX only — the API's token check on every request is the real security.
export default function RequireAuth() {
  // Hooks first, before any return: React needs them in the same order on every render.
  const [status, setStatus] = useState<Status>('checking')
  const location = useLocation()

  useEffect(() => {
    let cancelled = false
    isSignedIn().then((ok) => {
      // Ignore a late answer if the user has already navigated away.
      if (!cancelled) setStatus(ok ? 'signed-in' : 'signed-out')
    })
    return () => {
      cancelled = true
    }
  }, []) // [] = check once, when the guard first appears

  if (status === 'checking') {
    // Show nothing protected while we wait: no flash of private content, no early API calls.
    return (
      <p role="status" className="text-sm text-stone-600">
        Checking your session…
      </p>
    )
  }

  if (status === 'signed-out') {
    // replace: swap this history entry, so Back doesn't bounce the user into a redirect loop.
    // state.from: sign-in can send them back here afterwards.
    return <Navigate to="/signin" replace state={{ from: location.pathname }} />
  }

  return <Outlet /> // the protected page
}