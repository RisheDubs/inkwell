import { useCallback, useEffect, useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router'
import Editor from '../components/Editor'
import TagInput from '../components/TagInput'
import { ApiError } from '../lib/api'
import { todayLocal } from '../lib/dates'
import {
  MAX_TITLE_LENGTH,
  MOODS,
  createEntry,
  deleteEntry,
  getEntry,
  updateEntry,
  type EntryInput,
  type EntryWithBody,
  type Mood,
  type TiptapDoc,
} from '../lib/entries'

const inputClass = 'mt-1 w-full rounded border border-stone-300 p-2'
const EMPTY_DOC: TiptapDoc = { type: 'doc', content: [{ type: 'paragraph' }] }

// Everything on the form except the body, which the editor owns.
type Fields = Omit<EntryInput, 'body'>

function fieldsOf(entry: EntryWithBody): Fields {
  return { title: entry.title, entryDate: entry.entryDate, mood: (entry.mood ?? '') as Mood | '', tags: entry.tags }
}

// Only what differs from the saved entry. PATCH leaves every other field alone, so an
// unchanged body is never re-sent (or accidentally overwritten).
function changedFields(saved: Fields, current: Fields): Partial<Fields> {
  const changes: Partial<Fields> = {}
  if (current.title !== saved.title) changes.title = current.title
  if (current.entryDate !== saved.entryDate) changes.entryDate = current.entryDate
  if (current.mood !== saved.mood) changes.mood = current.mood
  if (current.tags.join(',') !== saved.tags.join(',')) changes.tags = current.tags
  return changes
}

// /entries/new and /entries/:id render the same component. The key makes React start fresh
// when the ID changes (after "Create", or moving between entries), instead of keeping old state.
export default function EntryPage() {
  const { id } = useParams()
  return <EntryForm key={id ?? 'new'} id={id} />
}

function EntryForm({ id }: { id?: string }) {
  const isNew = id === undefined
  const navigate = useNavigate()
  const location = useLocation()
  const justCreated = (location.state as { created?: boolean } | null)?.created === true

  const [saved, setSaved] = useState<EntryWithBody | null>(null) // the version the API last returned
  const [fields, setFields] = useState<Fields>({ title: '', entryDate: todayLocal(), mood: '', tags: [] })
  const [body, setBody] = useState<TiptapDoc | undefined>() // only set once the user edits the body
  const [bodyLocked, setBodyLocked] = useState(false)
  const lockBody = useCallback(() => setBodyLocked(true), []) // stable, so Editor's effect runs once
  const [status, setStatus] = useState<'loading' | 'ready' | 'saving' | 'deleting' | 'not-found'>(
    isNew ? 'ready' : 'loading',
  )
  const [confirmingDelete, setConfirmingDelete] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState(justCreated ? 'Entry created.' : '')

  const handleError = useCallback(
    (err: unknown, action: 'load' | 'save' | 'delete') => {
      if (err instanceof ApiError && err.status === 401) {
        navigate('/signin', { replace: true, state: { from: location.pathname } })
        return
      }
      if (err instanceof ApiError && err.status === 404) {
        setStatus('not-found') // missing, deleted, or someone else's: the API answers 404 for all three
        return
      }
      setStatus('ready')
      if (err instanceof ApiError && err.status === 400) {
        setError(err.message) // the API's validation message is written for people
        return
      }
      const ref = err instanceof ApiError && err.requestId ? ` (reference: ${err.requestId})` : ''
      setError(`We couldn't ${action} this entry. Please try again.${ref}`)
    },
    [navigate, location.pathname],
  )

  useEffect(() => {
    if (id === undefined) return
    let cancelled = false
    getEntry(id)
      .then((entry) => {
        if (cancelled) return
        setSaved(entry)
        setFields(fieldsOf(entry))
        setStatus('ready')
      })
      .catch((err: unknown) => {
        if (!cancelled) handleError(err, 'load')
      })
    return () => {
      cancelled = true
    }
  }, [id, handleError])

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setNotice('')
    const current = { ...fields, title: fields.title.trim() }

    try {
      if (saved === null) {
        setStatus('saving')
        const created = await createEntry({ ...current, body: body ?? EMPTY_DOC })
        navigate(`/entries/${created.id}`, { replace: true, state: { created: true } })
        return
      }

      const changes: Partial<EntryInput> = changedFields(fieldsOf(saved), current)
      if (body && !bodyLocked) changes.body = body
      if (Object.keys(changes).length === 0) {
        setNotice('Nothing to save.')
        return
      }

      setStatus('saving')
      const updated = await updateEntry(saved.id, changes)
      setSaved(updated)
      setFields(fieldsOf(updated))
      setBody(undefined)
      setNotice('Saved.')
      setStatus('ready')
    } catch (err) {
      handleError(err, 'save')
    }
  }

  async function handleDelete() {
    if (saved === null) return
    setError('')
    setNotice('')
    setStatus('deleting')
    try {
      await deleteEntry(saved.id)
      // replace: Back must not return to the page of an entry that no longer exists.
      navigate('/', { replace: true })
    } catch (err) {
      setConfirmingDelete(false)
      handleError(err, 'delete')
    }
  }

  if (status === 'loading') {
    return (
      <p role="status" className="text-sm text-stone-500">
        Loading entry…
      </p>
    )
  }

  if (status === 'not-found') {
    return (
      <div>
        <h1 className="text-2xl font-semibold">Entry not found</h1>
        <p className="mt-1 text-stone-600">It may have been deleted, or the link may be wrong.</p>
        <Link to="/" className="mt-2 inline-block underline">
          Back to your journal
        </Link>
      </div>
    )
  }

  return (
    <div className="max-w-2xl">
      <Link to="/" className="text-sm underline">
        ← Your journal
      </Link>
      <h1 className="mt-2 text-2xl font-semibold">{isNew ? 'New entry' : 'Edit entry'}</h1>

      <form onSubmit={handleSubmit} className="mt-4 space-y-4">
        <label className="block">
          <span className="text-sm">Title</span>
          <input
            required
            maxLength={MAX_TITLE_LENGTH}
            value={fields.title}
            onChange={(e) => setFields((f) => ({ ...f, title: e.target.value }))}
            className={inputClass}
          />
        </label>

        <div className="flex flex-wrap gap-4">
          <label className="block">
            <span className="text-sm">Date</span>
            <input
              type="date"
              required
              min="1900-01-01"
              value={fields.entryDate}
              onChange={(e) => setFields((f) => ({ ...f, entryDate: e.target.value }))}
              className={inputClass}
            />
          </label>
          <label className="block">
            <span className="text-sm">Mood</span>
            <select
              value={fields.mood}
              onChange={(e) => setFields((f) => ({ ...f, mood: e.target.value as Mood | '' }))}
              className={inputClass}
            >
              <option value="">No mood</option>
              {MOODS.map((mood) => (
                <option key={mood} value={mood}>
                  {mood}
                </option>
              ))}
            </select>
          </label>
        </div>

        <TagInput tags={fields.tags} onChange={(tags) => setFields((f) => ({ ...f, tags }))} />

        <Editor
          label="Entry"
          initialContent={saved?.body}
          onChange={setBody}
          onContentError={lockBody}
        />
        {bodyLocked && (
          <p role="alert" className="text-sm text-red-700">
            This entry contains something the editor can't display yet, so the text is locked to keep it
            safe. You can still change the title, date, mood and tags.
          </p>
        )}

        {error && (
          <p role="alert" className="text-sm text-red-700">
            {error}
          </p>
        )}
        {notice && (
          <p role="status" className="text-sm text-green-700">
            {notice}
          </p>
        )}

        <button
          type="submit"
          disabled={status === 'saving' || status === 'deleting'}
          className="rounded bg-stone-900 px-4 py-2 text-white disabled:opacity-50"
        >
          {status === 'saving' ? 'Saving…' : isNew ? 'Create entry' : 'Save changes'}
        </button>
      </form>

      {/* Outside the form, so these buttons can never submit it. */}
      {saved && (
        <div className="mt-8 border-t border-stone-200 pt-4">
          {confirmingDelete ? (
            <div role="group" aria-label="Confirm delete" className="space-y-2">
              <p className="text-sm">Delete “{saved.title}”? It will disappear from your journal.</p>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  onClick={() => void handleDelete()}
                  disabled={status === 'deleting'}
                  className="rounded bg-red-700 px-4 py-2 text-sm text-white disabled:opacity-50"
                >
                  {status === 'deleting' ? 'Deleting…' : 'Yes, delete'}
                </button>
                <button type="button" onClick={() => setConfirmingDelete(false)} className="text-sm underline">
                  Cancel
                </button>
              </div>
            </div>
          ) : (
            <button type="button" onClick={() => setConfirmingDelete(true)} className="text-sm text-red-700 underline">
              Delete entry
            </button>
          )}
        </div>
      )}
    </div>
  )
}