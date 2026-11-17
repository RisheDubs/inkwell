import { Fragment, useCallback, useEffect, useState } from 'react'
import { Outlet, useParams } from 'react-router'
import { ImageResolverContext } from '../lib/imageResolver'
import { getPublicImageUrl, getPublicJournal, type PublicJournal } from '../lib/public'
import { themeStyle } from '../lib/theme'
import PublicNotFound from './PublicNotFound'

// What the public pages inside the frame read with useOutletContext<PublicContext>().
export type PublicContext = { journal: PublicJournal }

type Loaded = { handle: string; journal?: PublicJournal; missing?: boolean }

// Wraps /j/{handle} and its entries: loads the public journal once, applies its theme, and
// chooses the public image resolver (ADR-0009). No token anywhere below this point.
// Unlike JournalFrame, a journal that can't load is the gate: nothing inside is shown.
export default function PublicFrame() {
  const { handle = '' } = useParams()
  const [loaded, setLoaded] = useState<Loaded>()

  useEffect(() => {
    let cancelled = false
    getPublicJournal(handle)
      .then((journal) => {
        if (!cancelled) setLoaded({ handle, journal })
      })
      .catch(() => {
        // Unknown, private, malformed or a server error: all look the same (one 404 on the API).
        if (!cancelled) setLoaded({ handle, missing: true })
      })
    return () => {
      cancelled = true
    }
  }, [handle])

  // Same function for the same handle, so image effects don't re-run on every render.
  const resolve = useCallback((mediaId: string) => getPublicImageUrl(handle, mediaId), [handle])

  // Only trust an answer for the handle in the URL now.
  const current = loaded?.handle === handle ? loaded : undefined

  if (!current) {
    return (
      <p role="status" className="text-sm text-stone-600">
        Loading journal…
      </p>
    )
  }
  if (!current.journal) return <PublicNotFound />

  const { className, style } = themeStyle(current.journal.theme)

  return (
    <ImageResolverContext value={resolve}>
      <div data-testid="journal-frame" className={`journal rounded p-4 ${className}`} style={style}>
        {/* key: a different journal starts its pages fresh (no entries carried over). */}
        <Fragment key={current.journal.handle}>
          <Outlet context={{ journal: current.journal } satisfies PublicContext} />
        </Fragment>
      </div>
    </ImageResolverContext>
  )
}