import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getImageUrl } from '../lib/media'
import { withImages } from '../test/images'
import CoverImage from './CoverImage'

vi.mock('../lib/media', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/media')>()),
  getImageUrl: vi.fn(),
}))

describe('CoverImage', () => {
  beforeEach(() => {
    vi.mocked(getImageUrl).mockReset()
  })

  it('shows the cover through a fresh link from the API', async () => {
    vi.mocked(getImageUrl).mockResolvedValue('https://s3.example/cover')
    render(withImages(<CoverImage mediaId="6af4937db325014a79b03c88.png" />))

    expect(await screen.findByRole('img', { name: 'Journal cover' })).toHaveAttribute('src', 'https://s3.example/cover')
  })

  it('draws nothing, and asks nothing, for something that is not a media ID', () => {
    const { container } = render(withImages(<CoverImage mediaId="https://tracker.example/pixel.gif" />))

    expect(container).toBeEmptyDOMElement()
    expect(getImageUrl).not.toHaveBeenCalled()
  })

  it('fails loudly when no page chose an image resolver (ADR-0009)', () => {
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    expect(() => render(<CoverImage mediaId="6af4937db325014a79b03c88.png" />)).toThrow(/ADR-0009/)
    expect(getImageUrl).not.toHaveBeenCalled()
    spy.mockRestore()
  })
})