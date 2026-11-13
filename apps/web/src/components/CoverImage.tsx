import { useEffect, useState } from 'react'
import { getImageUrl, isMediaId } from '../lib/media'

type Props = { mediaId: string; className?: string }

// The journal's cover. Like images in entries, it is stored as a media ID and shown through a
// fresh short-lived link from our API. If it can't be shown, nothing is drawn.
export default function CoverImage({ mediaId, className = '' }: Props) {
  const [loaded, setLoaded] = useState<{ id: string; url?: string }>()

  useEffect(() => {
    if (!isMediaId(mediaId)) return
    let cancelled = false
    getImageUrl(mediaId)
      .then((url) => {
        if (!cancelled) setLoaded({ id: mediaId, url })
      })
      .catch(() => {
        if (!cancelled) setLoaded({ id: mediaId })
      })
    return () => {
      cancelled = true
    }
  }, [mediaId])

  const url = loaded?.id === mediaId ? loaded.url : undefined
  if (!url) return null
  return <img src={url} alt="Journal cover" className={`w-full rounded object-cover ${className}`} />
}