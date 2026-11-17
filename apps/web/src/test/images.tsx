import type { ReactElement, ReactNode } from 'react'
import { ImageResolverContext } from '../lib/imageResolver'
import { getImageUrl } from '../lib/media'

// Tests render components outside the real router, so nothing has chosen an image resolver
// (ADR-0009). This gives them the signed-in one. In a test that mocks '../lib/media',
// getImageUrl here is that mock, so existing assertions on it keep working.
export function withImages(ui: ReactNode): ReactElement {
  return <ImageResolverContext value={getImageUrl}>{ui}</ImageResolverContext>
}