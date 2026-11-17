import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { ImageResolverContext } from '../lib/imageResolver'
import ReadOnlyBody from './ReadOnlyBody'

const MEDIA_ID = '6af4937db325014a79b03c88.png'

const doc = (...content: object[]) => ({ type: 'doc', content })
const text = (t: string) => ({ type: 'paragraph', content: [{ type: 'text', text: t }] })

describe('ReadOnlyBody', () => {
  it('shows the text, and nothing can be edited', async () => {
    render(<ReadOnlyBody label="Entry text" content={doc(text('Hello there'))} />)

    const body = await screen.findByLabelText('Entry text')
    expect(body).toHaveTextContent('Hello there')
    expect(body).toHaveAttribute('contenteditable', 'false')
    expect(screen.queryByLabelText('Choose an image')).not.toBeInTheDocument() // no toolbar
  })

  it('asks the page’s resolver for image links', async () => {
    const resolve = vi.fn(async () => 'https://s3.example/public-link')
    render(
      <ImageResolverContext value={resolve}>
        <ReadOnlyBody label="Entry text" content={doc(text('Look:'), { type: 'mediaImage', attrs: { mediaId: MEDIA_ID, alt: '' } })} />
      </ImageResolverContext>,
    )

    expect(await screen.findByRole('img', { name: 'Image in this entry' })).toHaveAttribute('src', 'https://s3.example/public-link')
    expect(resolve).toHaveBeenCalledWith(MEDIA_ID)
  })

  it('refuses content it does not know, instead of showing part of it', async () => {
    const spy = vi.spyOn(console, 'warn').mockImplementation(() => {})
    render(<ReadOnlyBody label="Entry text" content={doc({ type: 'script', content: [{ type: 'text', text: 'x' }] })} />)

    expect(await screen.findByText('This entry can’t be shown.')).toBeInTheDocument()
    expect(screen.queryByLabelText('Entry text')).not.toBeInTheDocument()
    spy.mockRestore()
  })
})