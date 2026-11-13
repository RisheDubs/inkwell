import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router'
import { getJournal, type Journal } from '../lib/journal'
import { themeStyle } from '../lib/theme'
import CoverImage from './CoverImage'

// What the pages inside the frame can read with useOutletContext<JournalContext>().
export type JournalContext = { journal: Journal | null; setJournal: (journal: Journal) => void }

// Wraps every signed-in page in the journal's theme. The theme only changes CSS variables and a
// paper class here, so no page needs to know which theme is active.
export default function JournalFrame() {
  const [journal, setJournal] = useState<Journal | null>(null)
  const { pathname } = useLocation()

  useEffect(() => {
    let cancelled = false
    getJournal()
      .then((loaded) => {
        if (!cancelled) setJournal(loaded)
      })
      .catch(() => {
        // The theme is decoration: if it can't load, keep the default look. The pages inside
        // make their own API calls and handle sign-in or errors themselves.
      })
    return () => {
      cancelled = true
    }
  }, [])

  const { className, style } = themeStyle(journal?.theme)

  return (
    <div data-testid="journal-frame" className={`journal rounded p-4 ${className}`} style={style}>
      {/* The cover heads the journal's home page, like the front of a notebook. */}
      {pathname === '/' && journal?.theme.coverMediaId && (
        <CoverImage mediaId={journal.theme.coverMediaId} className="mb-4 h-32" />
      )}
      <Outlet context={{ journal, setJournal } satisfies JournalContext} />
    </div>
  )
}