import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router'
import { ApiError, getMe, type Me } from '../lib/api'

// The journal home. For now it proves the full chain works: token, proxy, Go API, MongoDB.
// Becomes the timeline in step 13.
export default function Home() {
  const navigate = useNavigate()
  const [me, setMe] = useState<Me | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    getMe()
      .then((user) => {
        if (!cancelled) setMe(user)
      })
      .catch((err: unknown) => {
        if (cancelled) return
        if (err instanceof ApiError && err.status === 401) {
          // The API is the source of truth: if it rejects the token, the session is over.
          navigate('/signin', { replace: true, state: { from: '/' } })
          return
        }
        // The request ID ties this message to the exact line in the API's logs.
        const ref = err instanceof ApiError && err.requestId ? ` (reference: ${err.requestId})` : ''
        setError(`We couldn't load your account. Please try again.${ref}`)
      })
    return () => {
      cancelled = true
    }
  }, [navigate])

  return (
    <div>
      <h1 className="text-2xl font-semibold">Your journal</h1>
      {me && <p className="mt-1 text-sm text-stone-600">Signed in as {me.email}</p>}
      {!me && !error && (
        <p role="status" className="mt-1 text-sm text-stone-500">
          Loading your account…
        </p>
      )}
      {error && (
        <p role="alert" className="mt-2 text-sm text-red-700">
          {error}
        </p>
      )}
    </div>
  )
}