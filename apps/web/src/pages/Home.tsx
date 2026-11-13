import { useCallback, useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router'
import EntryCard from '../components/EntryCard'
import { ApiError } from '../lib/api'
import { listEntries, type Entry } from '../lib/entries'

const PAGE_SIZE = 20

// The journal timeline: newest first, "Load more" via the API's cursor, filter by tag.
export default function Home() {
  const navigate = useNavigate()
  const [entries, setEntries] = useState<Entry[]>([])
  const [nextCursor, setNextCursor] = useState<string | undefined>()
  const [tag, setTag] = useState<string | undefined>()
  const [status, setStatus] = useState<'loading' | 'ready' | 'loading-more'>('loading')
  const [error, setError] = useState('')
  // Changes every time the list starts over, so a late answer for an old list is ignored.
  const generation = useRef(0)

  const handleError = useCallback(
    (err: unknown) => {
      if (err instanceof ApiError && err.status === 401) {
        // The API is the source of truth: if it rejects the token, the session is over.
        navigate('/signin', { replace: true, state: { from: '/' } })
        return
      }
      const ref = err instanceof ApiError && err.requestId ? ` (reference: ${err.requestId})` : ''
      setError(`We couldn't load your entries. Please try again.${ref}`)
      setStatus('ready')
    },
    [navigate],
  )

  // First page, and again whenever the tag filter changes.
  useEffect(() => {
    const current = ++generation.current
    listEntries({ tag, limit: PAGE_SIZE })
      .then((page) => {
        if (current !== generation.current) return // a newer list has started: drop this answer
        setEntries(page.entries)
        setNextCursor(page.nextCursor)
        setStatus('ready')
      })
      .catch((err: unknown) => {
        if (current === generation.current) handleError(err)
      })
  }, [tag, handleError])

  // Reset the list in the click handler (not inside the effect), then let the effect fetch.
  function chooseTag(next: string | undefined) {
    if (next === tag) return
    setEntries([])
    setNextCursor(undefined)
    setError('')
    setStatus('loading')
    setTag(next)
  }

  async function loadMore() {
    if (!nextCursor) return
    const current = generation.current
    setStatus('loading-more')
    try {
      const page = await listEntries({ tag, cursor: nextCursor, limit: PAGE_SIZE })
      if (current !== generation.current) return // the filter changed while we waited
      setEntries((previous) => [...previous, ...page.entries])
      setNextCursor(page.nextCursor)
      setStatus('ready')
    } catch (err) {
      if (current === generation.current) handleError(err)
    }
  }

  return (
    <div>
      <h1 className="text-2xl font-semibold">Your journal</h1>

      {tag && (
        <p className="mt-2 text-sm">
          Showing entries tagged <strong>#{tag}</strong>{' '}
          <button type="button" onClick={() => chooseTag(undefined)} className="underline">
            Show all
          </button>
        </p>
      )}

      {status === 'loading' && (
        <p role="status" className="mt-4 text-sm text-stone-500">
          Loading your entries…
        </p>
      )}

      {error && (
        <p role="alert" className="mt-4 text-sm text-red-700">
          {error}
        </p>
      )}

      {status !== 'loading' && !error && entries.length === 0 && (
        <p className="mt-4 text-stone-600">{tag ? `No entries tagged #${tag}.` : 'No entries yet.'}</p>
      )}

      <ul className="mt-4 space-y-3">
        {entries.map((entry) => (
          <li key={entry.id}>
            <EntryCard entry={entry} onTagClick={chooseTag} />
          </li>
        ))}
      </ul>

      {nextCursor && (
        <button
          type="button"
          onClick={() => void loadMore()}
          disabled={status === 'loading-more'}
          className="mt-4 rounded border border-stone-300 px-4 py-2 text-sm disabled:opacity-50"
        >
          {status === 'loading-more' ? 'Loading…' : 'Load more'}
        </button>
      )}
    </div>
  )
}