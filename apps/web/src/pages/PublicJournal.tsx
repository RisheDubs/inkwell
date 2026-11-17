import { useState } from 'react'
import { useOutletContext } from 'react-router'
import CoverImage from '../components/CoverImage'
import EntryCard from '../components/EntryCard'
import type { PublicContext } from '../components/PublicFrame'
import { getPublicJournal, type PublicEntrySummary } from '../lib/public'

// /j/{handle}: a public journal's front page. Read-only; the frame has already loaded the
// journal and its first page of entries.
export default function PublicJournal() {
  const { journal } = useOutletContext<PublicContext>()
  const [more, setMore] = useState<PublicEntrySummary[]>([])
  const [cursor, setCursor] = useState(journal.nextCursor)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  async function loadMore() {
    if (!cursor) return
    setLoading(true)
    setError('')
    try {
      const next = await getPublicJournal(journal.handle, { cursor })
      setMore((shown) => [...shown, ...next.entries])
      setCursor(next.nextCursor)
    } catch {
      setError('Couldn’t load more entries. Please try again.')
    } finally {
      setLoading(false)
    }
  }

  const entries = [...journal.entries, ...more]
  const base = `/j/${encodeURIComponent(journal.handle)}`

  return (
    <>
      {journal.theme.coverMediaId && <CoverImage mediaId={journal.theme.coverMediaId} className="mb-4 h-32" />}
      <h1 className="text-2xl font-semibold">{journal.title}</h1>
      <p className="mt-1 text-sm text-stone-500">A public journal on Inkwell</p>

      {entries.length === 0 ? (
        <p className="mt-4 text-stone-600">No entries yet.</p>
      ) : (
        <div className="mt-4 space-y-3">
          {entries.map((entry) => (
            <EntryCard key={entry.id} entry={entry} to={`${base}/${encodeURIComponent(entry.id)}`} />
          ))}
        </div>
      )}

      {cursor && (
        <button
          type="button"
          onClick={() => void loadMore()}
          disabled={loading}
          className="mt-4 rounded border border-stone-300 px-3 py-1 text-sm disabled:opacity-50"
        >
          {loading ? 'Loading…' : 'Load more'}
        </button>
      )}
      {error && (
        <p role="alert" className="mt-2 text-sm text-red-700">
          {error}
        </p>
      )}
    </>
  )
}