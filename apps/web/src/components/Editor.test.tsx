import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import Editor from './Editor'

const doc = (text: string) => ({
  type: 'doc' as const,
  content: [{ type: 'paragraph', content: [{ type: 'text', text }] }],
})

describe('Editor', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows the stored body', () => {
    render(<Editor label="Entry" initialContent={doc('Hello there')} onChange={() => {}} />)
    expect(screen.getByRole('textbox', { name: 'Entry' })).toHaveTextContent('Hello there')
  })

  it('reports what the user types as Tiptap JSON', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(<Editor label="Entry" onChange={onChange} />)

    await user.click(screen.getByRole('textbox', { name: 'Entry' }))
    await user.keyboard('Hi')

    expect(onChange).toHaveBeenLastCalledWith(doc('Hi'))
  })

  it('formats with the toolbar and marks the button as pressed', async () => {
    const onChange = vi.fn()
    const user = userEvent.setup()
    render(<Editor label="Entry" initialContent={doc('Item')} onChange={onChange} />)

    await user.click(screen.getByRole('button', { name: 'Bullet list' }))

    expect(onChange).toHaveBeenLastCalledWith(
      expect.objectContaining({ content: expect.arrayContaining([expect.objectContaining({ type: 'bulletList' })]) }),
    )
    expect(screen.getByRole('button', { name: 'Bullet list' })).toHaveAttribute('aria-pressed', 'true')
  })

  it('shows HTML-looking text as text, never as HTML', () => {
    const { container } = render(
      <Editor label="Entry" initialContent={doc('<img src=x onerror=alert(1)>')} onChange={() => {}} />,
    )
    expect(screen.getByRole('textbox', { name: 'Entry' })).toHaveTextContent('<img src=x onerror=alert(1)>')
    expect(container.querySelector('img')).toBeNull()
  })

  it('locks itself and reports content it cannot show, instead of showing an empty body', () => {
    vi.spyOn(console, 'warn').mockImplementation(() => {}) // Tiptap warns about the bad content; expected here
    const onContentError = vi.fn()
    const unknown = { type: 'doc' as const, content: [{ type: 'script', content: [{ type: 'text', text: 'x' }] }] }
    render(<Editor label="Entry" initialContent={unknown} onChange={() => {}} onContentError={onContentError} />)

    expect(onContentError).toHaveBeenCalledOnce()
    expect(screen.getByRole('textbox', { name: 'Entry' })).toHaveAttribute('contenteditable', 'false')
    expect(screen.getByRole('button', { name: 'Bold' })).toBeDisabled()
  })
})