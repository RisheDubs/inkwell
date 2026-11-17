import { useId, useState, type KeyboardEvent } from 'react'
import { MAX_TAGS, normaliseTag } from '../lib/entries'

type Props = { tags: string[]; onChange: (tags: string[]) => void }

// Type a tag and press Enter or comma. Tags are normalised the way the API expects
// ("Work Life" -> "work-life"); anything the API would reject is explained, not sent.
export default function TagInput({ tags, onChange }: Props) {
  const [draft, setDraft] = useState('')
  const [hint, setHint] = useState('')
  const hintId = useId()

  function add(raw: string) {
    if (!raw.trim()) return
    const tag = normaliseTag(raw)
    if (!tag) {
      setHint('Tags can use a-z, 0-9 and hyphens, up to 32 characters.')
      return
    }
    if (tags.includes(tag)) {
      setDraft('') // already there: nothing to add
      setHint('')
      return
    }
    if (tags.length >= MAX_TAGS) {
      setHint(`An entry can have at most ${MAX_TAGS} tags.`)
      return
    }
    onChange([...tags, tag])
    setDraft('')
    setHint('')
  }

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === 'Enter' || event.key === ',') {
      event.preventDefault() // Enter would otherwise submit the whole entry form
      add(draft)
    } else if (event.key === 'Backspace' && draft === '' && tags.length > 0) {
      onChange(tags.slice(0, -1)) // Backspace in an empty box removes the last tag
    }
  }

  return (
    <div>
      <label className="block">
        <span className="text-sm">Tags</span>
        <input
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={handleKeyDown}
          onBlur={() => add(draft)}
          placeholder="Type a tag, then press Enter"
          aria-describedby={hint ? hintId : undefined}
          className="mt-1 w-full rounded border border-stone-300 p-2"
        />
      </label>
      {hint && (
        <p id={hintId} role="alert" className="mt-1 text-sm text-red-700">
          {hint}
        </p>
      )}
      {tags.length > 0 && (
        <ul aria-label="Tags" className="mt-2 flex flex-wrap gap-2">
          {tags.map((tag) => (
            <li key={tag} className="flex items-center gap-1 rounded bg-stone-100 px-2 py-0.5 text-sm">
              #{tag}
              <button
                type="button"
                aria-label={`Remove tag ${tag}`}
                onClick={() => onChange(tags.filter((t) => t !== tag))}
                className="text-stone-500 hover:text-stone-900"
              >
                ×
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  )
}