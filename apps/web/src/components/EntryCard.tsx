import { formatEntryDate } from '../lib/dates'
import type { Entry } from '../lib/entries'

type Props = { entry: Entry; onTagClick: (tag: string) => void }

// One entry in the timeline. Display only: data comes in as props, clicks go out as callbacks.
// All text is rendered with {…}, which React escapes, so stored text can never run as script.
export default function EntryCard({ entry, onTagClick }: Props) {
  return (
    <article className="rounded border border-stone-200 bg-white p-4">
      <header className="flex items-baseline justify-between gap-4">
        <h2 className="font-semibold">{entry.title || 'Untitled'}</h2>
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
        {entry.tags.map((tag) => (
          <button
            key={tag}
            type="button"
            onClick={() => onTagClick(tag)}
            className="rounded bg-stone-100 px-2 py-0.5 text-stone-700 hover:bg-stone-200"
          >
            #{tag}
          </button>
        ))}
      </footer>
    </article>
  )
}