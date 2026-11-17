import { useEffect, useState } from 'react'
import { Link, useOutletContext, useParams } from 'react-router'
import PublicNotFound from '../components/PublicNotFound'
import type { PublicContext } from '../components/PublicFrame'
import ReadOnlyBody from '../components/ReadOnlyBody'
import { formatEntryDate } from '../lib/dates'
import { getPublicEntry, type PublicEntry as Entry } from '../lib/public'

type Loaded = { id: string; entry?: Entry; missing?: boolean }

// /j/{handle}/{id}: one entry of a public journal, read-only.
export default function PublicEntry() {
  const { journal } = useOutletContext<PublicContext>()
  const { id = '' } = useParams()
  const [loaded, setLoaded] = useState<Loaded>()

  useEffect(() => {
    let cancelled = false
    getPublicEntry(journal.handle, id)
      .then((entry) => {
        if (!cancelled) setLoaded({ id, entry })
      })
      .catch(() => {
        if (!cancelled) setLoaded({ id, missing: true })
      })
    return () => {
      cancelled = true
    }
  }, [journal.handle, id])

  const current = loaded?.id === id ? loaded : undefined
  if (!current) {
    return (
      <p role="status" className="text-sm text-stone-600">
        Loading entry…
      </p>
    )
  }
  if (!current.entry) return <PublicNotFound />

  const entry = current.entry
  return (
    <article>
      <Link to={`/j/${encodeURIComponent(journal.handle)}`} className="text-sm underline">
        ← {journal.title}
      </Link>
      <h1 className="mt-2 text-2xl font-semibold">{entry.title || 'Untitled'}</h1>
      <p className="mt-1 flex flex-wrap gap-2 text-sm text-stone-500">
        <time dateTime={entry.entryDate}>{formatEntryDate(entry.entryDate)}</time>
        {entry.mood && <span>Mood: {entry.mood}</span>}
        {entry.tags.map((tag) => (
          <span key={tag}>#{tag}</span>
        ))}
      </p>
      <div className="mt-4 rounded border border-stone-300 bg-white">
        {/* key: a different entry gets a fresh read-only editor (content is read once). */}
        <ReadOnlyBody key={entry.id} label="Entry text" content={entry.body} />
      </div>
    </article>
  )
}