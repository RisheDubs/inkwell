import { useEffect, useId, useState, type FormEvent } from 'react'
import { useLocation, useNavigate } from 'react-router'
import { ApiError, getMe, updateMe, type Me } from '../lib/api'
import { HANDLE_RULES, handleProblem, normalizeHandle, publicPath, publicUrl } from '../lib/handle'
import { updateJournal, type Journal } from '../lib/journal'

type Props = { journal: Journal; onSaved: (journal: Journal) => void }

// Handle and visibility. Saves straight away, separately from the theme form.
// The handle lives on the user (PATCH /api/me); visibility on the journal (PATCH /api/journal).
export default function SharingSettings({ journal, onSaved }: Props) {
  const navigate = useNavigate()
  const location = useLocation()
  const handleId = useId()
  const [me, setMe] = useState<Me | null>(null)
  const [loadFailed, setLoadFailed] = useState(false)
  const [handle, setHandle] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  useEffect(() => {
    let cancelled = false
    getMe()
      .then((loaded) => {
        if (cancelled) return
        setMe(loaded)
        setHandle(loaded.handle ?? '')
      })
      .catch(() => {
        if (!cancelled) setLoadFailed(true)
      })
    return () => {
      cancelled = true
    }
  }, [])

  // One place for busy state and errors; action returns the message to show on success.
  async function run(action: () => Promise<string>) {
    setError('')
    setNotice('')
    setBusy(true)
    try {
      setNotice(await action())
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate('/signin', { replace: true, state: { from: location.pathname } })
        return
      }
      if (err instanceof ApiError && err.status === 409) {
        setError('That handle is taken. Try another.')
      } else if (err instanceof ApiError && err.status === 400) {
        setError(err.message)
      } else {
        const ref = err instanceof ApiError && err.requestId ? ` (reference: ${err.requestId})` : ''
        setError(`We couldn't save your sharing settings. Please try again.${ref}`)
      }
    } finally {
      setBusy(false)
    }
  }

  if (loadFailed) {
    return <p className="text-sm text-red-700">We couldn't load your sharing settings. Reload to try again.</p>
  }
  if (!me) {
    return <p className="text-sm text-stone-500">Loading sharing settings…</p>
  }

  const saved = me.handle ?? ''
  const isPublic = journal.visibility === 'public'

  function saveHandle(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const next = normalizeHandle(handle)
    const problem = handleProblem(next)
    if (problem) {
      setNotice('')
      setError(problem)
      return
    }
    if (next === saved) {
      setError('')
      setNotice('That’s already your handle.')
      return
    }
    // A freed handle can be claimed by anyone, who would then appear at your old link.
    if (
      saved &&
      isPublic &&
      !window.confirm(`Your link ${publicPath(saved)} will stop working, and someone else could take “${saved}”. Change it?`)
    ) {
      return
    }
    void run(async () => {
      const updated = await updateMe({ handle: next })
      setMe(updated)
      setHandle(updated.handle ?? '')
      return 'Handle saved.'
    })
  }

  function setVisibility(makePublic: boolean) {
    if (
      makePublic &&
      !window.confirm(
        'Anyone with the link will be able to read this journal: its title, look, cover and every entry. Make it public?',
      )
    ) {
      return
    }
    void run(async () => {
      onSaved(await updateJournal({ visibility: makePublic ? 'public' : 'private' }))
      return makePublic ? 'Your journal is public.' : 'Your journal is private.'
    })
  }

  return (
    <section aria-labelledby="sharing-heading" className="space-y-3">
      <h2 id="sharing-heading" className="text-sm font-semibold">
        Sharing
      </h2>

      <form onSubmit={saveHandle} className="space-y-1">
        <label htmlFor={handleId} className="block text-sm">
          Handle
        </label>
        <div className="mt-1 flex items-center gap-1">
          {/* Shown, not part of the name: the field holds only the handle. */}
          <span aria-hidden="true" className="text-sm text-stone-500">
            /j/
          </span>
          <input
            id={handleId}
            value={handle}
            onChange={(e) => setHandle(e.target.value)}
            maxLength={30}
            autoComplete="off"
            spellCheck={false}
            aria-describedby={`${handleId}-rules`}
            className="w-full rounded border border-stone-300 p-2"
          />
        </div>
        <p id={`${handleId}-rules`} className="text-xs text-stone-500">
          {HANDLE_RULES}
        </p>
        <button
          type="submit"
          disabled={busy}
          className="rounded border border-stone-300 px-3 py-1.5 text-sm disabled:opacity-50"
        >
          {busy ? 'Saving…' : 'Save handle'}
        </button>
      </form>

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          checked={isPublic}
          disabled={busy || !saved}
          onChange={(e) => setVisibility(e.target.checked)}
        />
        Share this journal publicly
      </label>
      {!saved && <p className="text-xs text-stone-500">Choose a handle first.</p>}

      {isPublic && saved && (
        <p className="text-sm">
          Public link:{' '}
          {/* noopener: the new tab can't reach back into this one through window.opener. */}
          <a href={publicPath(saved)} target="_blank" rel="noopener noreferrer" className="underline">
            {publicUrl(saved)}
          </a>
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
    </section>
  )
}