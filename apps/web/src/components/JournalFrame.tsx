import { useCallback, useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router'
import { getMe } from '../lib/api'
import { getJournal, type Journal } from '../lib/journal'
import { themeStyle } from '../lib/theme'
import CoverImage from './CoverImage'

// What the pages inside the frame can read with useOutletContext<JournalContext>().
// loadFailed and retry are optional, so test stubs that only hold a journal still fit.
export type JournalContext = {
  journal: Journal | null
  setJournal: (journal: Journal) => void
  loadFailed?: boolean
  retry?: () => void
}

// Wraps every signed-in page in the journal's theme. The theme only changes CSS variables and a
// paper class here, so no page needs to know which theme is active.
export default function JournalFrame() {
  const [journal, setJournal] = useState<Journal | null>(null)
  const [loadFailed, setLoadFailed] = useState(false)
  const [attempt, setAttempt] = useState(0)
  const { pathname } = useLocation()

  useEffect(() => {
    let cancelled = false
    // GET /api/me first: on a new user's first visit it creates the user and their journal.
    // Asking for /api/journal straight away got a 404 for every new user (found in step 21).
    // If /api/me itself fails, still try the journal: its own error is the one that matters.
    getMe()
      .catch(() => undefined)
      .then(() => getJournal())
      .then((loaded) => {
        if (!cancelled) setJournal(loaded)
      })
      .catch(() => {
        // The theme is decoration: keep the default look. Pages that need the journal
        // (Settings) show the failure with a retry; the others make their own API calls.
        if (!cancelled) setLoadFailed(true)
      })
    return () => {
      cancelled = true
    }
  }, [attempt])

  const retry = useCallback(() => {
    setLoadFailed(false)
    setAttempt((n) => n + 1) // a new attempt re-runs the effect above
  }, [])

  const { className, style } = themeStyle(journal?.theme)

  return (
    <div data-testid="journal-frame" className={`journal rounded p-4 ${className}`} style={style}>
      {/* The cover heads the journal's home page, like the front of a notebook. */}
      {pathname === '/' && journal?.theme.coverMediaId && (
        <CoverImage mediaId={journal.theme.coverMediaId} className="mb-4 h-32" />
      )}
      <Outlet context={{ journal, setJournal, loadFailed, retry } satisfies JournalContext} />
    </div>
  )
}