import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'
import TagInput from './TagInput'

// TagInput is controlled (the parent owns the tags), so test it inside a tiny parent.
function Harness({ initial = [], onSubmit = () => {} }: { initial?: string[]; onSubmit?: () => void }) {
  const [tags, setTags] = useState(initial)
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault()
        onSubmit()
      }}
    >
      <TagInput tags={tags} onChange={setTags} />
      <output>{tags.join(',')}</output>
    </form>
  )
}

const tagBox = () => screen.getByRole('textbox', { name: 'Tags' })

describe('TagInput', () => {
  it('adds a normalised tag on Enter without submitting the form', async () => {
    const onSubmit = vi.fn()
    const user = userEvent.setup()
    render(<Harness onSubmit={onSubmit} />)

    await user.type(tagBox(), 'Work Life{Enter}')

    expect(screen.getByRole('status')).toHaveTextContent('work-life')
    expect(tagBox()).toHaveValue('')
    expect(onSubmit).not.toHaveBeenCalled()
  })

  it('adds a tag on comma too, and ignores duplicates', async () => {
    const user = userEvent.setup()
    render(<Harness initial={['work']} />)

    await user.type(tagBox(), 'travel,work,')

    expect(screen.getByRole('status')).toHaveTextContent('work,travel')
  })

  it('explains a tag the API would reject, and adds nothing', async () => {
    const user = userEvent.setup()
    render(<Harness />)

    await user.type(tagBox(), 'café{Enter}')

    expect(screen.getByRole('alert')).toHaveTextContent('a-z, 0-9 and hyphens')
    expect(screen.getByRole('status')).toHaveTextContent('')
  })

  it('stops at the API limit of 20 tags', async () => {
    const user = userEvent.setup()
    render(<Harness initial={Array.from({ length: 20 }, (_, i) => `tag-${i}`)} />)

    await user.type(tagBox(), 'one-more{Enter}')

    expect(screen.getByRole('alert')).toHaveTextContent('at most 20 tags')
  })

  it('removes a tag with its button, or the last one with Backspace', async () => {
    const user = userEvent.setup()
    render(<Harness initial={['work', 'travel', 'family']} />)

    await user.click(screen.getByRole('button', { name: 'Remove tag travel' }))
    expect(screen.getByRole('status')).toHaveTextContent('work,family')

    await user.click(tagBox())
    await user.keyboard('{Backspace}')
    expect(screen.getByRole('status')).toHaveTextContent('work')
  })
})