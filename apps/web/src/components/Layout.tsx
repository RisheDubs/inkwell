import { useEffect, useState } from 'react'
import { Link, Outlet, useLocation, useNavigate } from 'react-router'
import { isSignedIn, signOutUser } from '../lib/auth'
import { clearImageUrlCache } from '../lib/media'

// Shared frame for every page: header with nav, then the current page in <Outlet />.
export default function Layout() {
  const location = useLocation()
  const navigate = useNavigate()
  const [signedIn, setSignedIn] = useState(false)

  // Re-check on every navigation, so the header updates right after sign-in or sign-out.
  // Cheap: Amplify answers from the tokens it already holds unless one needs refreshing.
  useEffect(() => {
    let cancelled = false
    isSignedIn().then((ok) => {
      if (!cancelled) setSignedIn(ok)
    })
    return () => {
      cancelled = true
    }
  }, [location.pathname])

  async function handleSignOut() {
    try {
      await signOutUser()
    } catch {
      // Even if Cognito can't be reached, still send the user back to sign-in.
    }
    clearImageUrlCache() // image links are signed for this user; the next person must not get them
    setSignedIn(false)
    navigate('/signin', { replace: true })
  }

  return (
    <div className="min-h-screen bg-stone-50 text-stone-900">
      <header className="border-b border-stone-200 bg-white">
        <nav className="mx-auto flex max-w-3xl items-center gap-4 p-4">
          <Link to="/" className="font-semibold text-red-600">Inkwell</Link>
          {signedIn ? (
            <>
              <Link to="/settings" className="ml-auto text-sm">Settings</Link>
              <button type="button" onClick={handleSignOut} className="text-sm underline">
                Sign out
              </button>
            </>
          ) : (
            <>
              <Link to="/signin" className="ml-auto text-sm">Sign in</Link>
              <Link to="/signup" className="text-sm">Sign up</Link>
            </>
          )}
        </nav>
      </header>
      <main className="mx-auto max-w-3xl p-4">
        <Outlet />
      </main>
    </div>
  )
}