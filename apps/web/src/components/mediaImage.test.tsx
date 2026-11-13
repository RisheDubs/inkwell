import { Editor as TiptapEditor } from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { getImageUrl, UploadError, uploadImage } from '../lib/media'
import Editor from './Editor'
import { MediaImage } from './mediaImage'

// Fake the network parts; keep the real isMediaId, IMAGE_TYPES and UploadError.
vi.mock('../lib/media', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../lib/media')>()),
  uploadImage: vi.fn(),
  getImageUrl: vi.fn(),
}))

const MEDIA_ID = '6af4937db325014a79b03c88.png'

const withImage = (mediaId: string) => ({
  type: 'doc' as const,
  content: [
    { type: 'paragraph', content: [{ type: 'text', text: 'Look:' }] },
    { type: 'mediaImage', attrs: { mediaId, alt: '' } },
  ],
})

describe('images in the editor', () => {
  beforeEach(() => {
    vi.mocked(uploadImage).mockReset()
    vi.mocked(getImageUrl).mockReset()
  })
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows a stored image using a fresh link from the API', async () => {
    vi.mocked(getImageUrl).mockResolvedValue('https://s3.example/fresh-link')
    render(<Editor label="Entry" initialContent={withImage(MEDIA_ID)} onChange={() => {}} />)

    expect(await screen.findByRole('img', { name: 'Image in this entry' })).toHaveAttribute(
      'src',
      'https://s3.example/fresh-link',
    )
    expect(getImageUrl).toHaveBeenCalledWith(MEDIA_ID)
  })

  it('shows "Image unavailable" for an ID that is not ours, without asking the API', async () => {
    render(<Editor label="Entry" initialContent={withImage('https://tracker.example/pixel.gif')} onChange={() => {}} />)

    // findBy: Tiptap draws image views a moment after the editor itself.
    expect(await screen.findByText('Image unavailable')).toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(getImageUrl).not.toHaveBeenCalled()
  })

  it('uploads a chosen image and stores only its media ID in the body', async () => {
    vi.mocked(uploadImage).mockResolvedValue(MEDIA_ID)
    vi.mocked(getImageUrl).mockResolvedValue('https://s3.example/fresh-link')
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(<Editor label="Entry" onChange={onChange} />)

    const photo = new File([new Uint8Array(10)], 'photo.png', { type: 'image/png' })
    await user.upload(screen.getByLabelText('Choose an image'), photo)

    expect(uploadImage).toHaveBeenCalledWith(photo)
    expect(await screen.findByRole('img', { name: 'Image in this entry' })).toHaveAttribute(
      'src',
      'https://s3.example/fresh-link',
    )
    const body = onChange.mock.lastCall?.[0]
    expect(body.content).toContainEqual({ type: 'mediaImage', attrs: { mediaId: MEDIA_ID, alt: '' } })
    expect(JSON.stringify(body)).not.toContain('https://') // no link is ever stored
  })

  it('explains a failed upload and inserts nothing', async () => {
    vi.mocked(uploadImage).mockImplementation(async () => {
      throw new UploadError('Images must be 5 MB or smaller.')
    })
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(<Editor label="Entry" onChange={onChange} />)

    await user.upload(screen.getByLabelText('Choose an image'), new File(['x'], 'big.png', { type: 'image/png' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('Images must be 5 MB or smaller.')
    expect(onChange).not.toHaveBeenCalled()
  })

  it('drops images pasted from other sites, but keeps our own', () => {
    const editor = new TiptapEditor({
      extensions: [StarterKit, MediaImage],
      content: `<p>Hi</p><img src="https://tracker.example/pixel.gif"><img data-media-id="${MEDIA_ID}">`,
    })

    expect(editor.getJSON().content).toEqual([
      { type: 'paragraph', content: [{ type: 'text', text: 'Hi' }] },
      { type: 'mediaImage', attrs: { mediaId: MEDIA_ID, alt: '' } },
    ])
    editor.destroy()
  })
})