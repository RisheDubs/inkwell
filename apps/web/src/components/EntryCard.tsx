import { Link } from 'react-router'
import { formatEntryDate } from '../lib/dates'

// What a card shows. Both a private Entry and a PublicEntrySummary fit, so the timeline and the
// public page share one card without the public side ever needing private fields.
type CardEntry = {
  id: string
  title: string
  entryDate: string
  mood?: string | null
  tags: readonly string[]
  excerpt?: string | null
  wordCount: number
}

type Props = {
  entry: CardEntry
  to?: string // where the title links; defaults to the signed-in entry page
  onTagClick?: (tag: string) => void // without it, tags are plain labels (the public page)
}

// One entry in the timeline. Display only: data comes in as props, clicks go out as callbacks.
// All text is rendered with {…}, which React escapes, so stored text can never run as script.
export default function EntryCard({ entry, to, onTagClick }: Props) {
  const href = to ?? `/entries/${encodeURIComponent(entry.id)}`
  return (
    <article className="rounded border border-stone-200 bg-white p-4">
      <header className="flex items-baseline justify-between gap-4">
        <h2 className="font-semibold">
          <Link to={href} className="hover:underline">
            {entry.title || 'Untitled'}
          </Link>
        </h2>
        <time dateTime={entry.entryDate} className="shrink-0 text-sm text-stone-500">
          {formatEntryDate(entry.entryDate)}
        </time>
      </header>
      {entry.excerpt && <p className="mt-2 text-sm text-stone-700">{entry.excerpt}</p>}
      <footer className="mt-3 flex flex-wrap items-center gap-2 text-xs text-stone-500">
        {entry.mood && <span>Mood: {entry.mood}</span>}
        <span>
          {entry.wordCount} {entry.wordCount === 1 ? 'word' : 'words'}
        </span>
        {entry.tags.map((tag) =>
          onTagClick ? (
            <button
              key={tag}
              type="button"
              onClick={() => onTagClick(tag)}
              className="rounded bg-stone-100 px-2 py-0.5 text-stone-700 hover:bg-stone-200"
            >
              #{tag}
            </button>
          ) : (
            <span key={tag} className="rounded bg-stone-100 px-2 py-0.5 text-stone-700">
              #{tag}
            </span>
          ),
        )}
      </footer>
    </article>
  )
}