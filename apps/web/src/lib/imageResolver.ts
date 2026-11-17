import { createContext, useContext } from 'react'

// Turns a stored media ID into a short-lived URL the browser can show.
// Signed-in pages use getImageUrl (token, own images); the public page uses
// getPublicImageUrl (no token, that journal's images). Components don't know which.
export type ImageResolver = (mediaId: string) => Promise<string>

// No default on purpose (ADR-0009): a page that forgets to choose fails loudly in tests,
// instead of quietly sending a visitor's token to the private route.
// Provide it with React 19's context-as-provider: <ImageResolverContext value={resolve}>.
export const ImageResolverContext = createContext<ImageResolver | null>(null)

export function useImageResolver(): ImageResolver {
  const resolve = useContext(ImageResolverContext)
  if (!resolve) {
    throw new Error('useImageResolver needs an <ImageResolverContext value={...}> above it (ADR-0009)')
  }
  return resolve
}