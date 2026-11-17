import { NodeViewWrapper, type ReactNodeViewProps } from '@tiptap/react'
import { useEffect, useState } from 'react'
import { useImageResolver } from '../lib/imageResolver'
import { isMediaId } from '../lib/media'

// How an image node looks in the editor. The node stores only a media ID; the picture's
// address is fetched fresh from our API, so the body never holds an expiring link or a URL
// that someone typed. Which API route depends on the page (ADR-0009): the signed-in pages
// use the private one, the public page the public one.
export default function MediaImageView({ node, selected }: ReactNodeViewProps) {
  const resolve = useImageResolver()
  const mediaId: unknown = node.attrs.mediaId
  const alt = typeof node.attrs.alt === 'string' ? node.attrs.alt : ''
  const [loaded, setLoaded] = useState<{ id: string; url?: string; failed?: boolean }>()

  useEffect(() => {
    if (!isMediaId(mediaId)) return
    let cancelled = false
    resolve(mediaId)
      .then((url) => {
        if (!cancelled) setLoaded({ id: mediaId, url })
      })
      .catch(() => {
        if (!cancelled) setLoaded({ id: mediaId, failed: true })
      })
    return () => {
      cancelled = true
    }
  }, [mediaId, resolve])

  // Only trust an answer for the ID we're showing now.
  const current = loaded?.id === mediaId ? loaded : undefined
  const unavailable = !isMediaId(mediaId) || current?.failed

  return (
    <NodeViewWrapper className={`my-2 ${selected ? 'rounded ring-2 ring-stone-400' : ''}`}>
      {unavailable ? (
        <p className="rounded border border-dashed border-stone-300 p-3 text-sm text-stone-500">Image unavailable</p>
      ) : current?.url ? (
        // A journal picture is content, not decoration, so it always gets a description.
        <img src={current.url} alt={alt || 'Image in this entry'} className="max-w-full rounded" />
      ) : (
        <p className="text-sm text-stone-500">Loading image…</p>
      )}
    </NodeViewWrapper>
  )
}