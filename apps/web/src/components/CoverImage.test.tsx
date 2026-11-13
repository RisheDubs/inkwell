import { render, screen } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getImageUrl } from '../lib/media'
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
    render(<CoverImage mediaId="6af4937db325014a79b03c88.png" />)

    expect(await screen.findByRole('img', { name: 'Journal cover' })).toHaveAttribute('src', 'https://s3.example/cover')
  })

  it('draws nothing, and asks nothing, for something that is not a media ID', () => {
    const { container } = render(<CoverImage mediaId="https://tracker.example/pixel.gif" />)

    expect(container).toBeEmptyDOMElement()
    expect(getImageUrl).not.toHaveBeenCalled()
  })
})