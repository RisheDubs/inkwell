import { useState, type FormEvent } from 'react'
import { Link, useLocation, useNavigate, useOutletContext } from 'react-router'
import type { JournalContext } from '../components/JournalFrame'
import { ApiError } from '../lib/api'
import {
  FONTS,
  MAX_JOURNAL_TITLE,
  PAPERS,
  updateJournal,
  type Font,
  type Journal,
  type JournalChanges,
  type Paper,
} from '../lib/journal'

const inputClass = 'mt-1 w-full rounded border border-stone-300 p-2'

type Form = { title: string; paper: Paper; font: Font; accent: string }

const formOf = (journal: Journal): Form => ({ title: journal.title, ...journal.theme })

// Only what differs from the saved journal: PATCH leaves the rest alone.
function changesBetween(saved: Form, current: Form): JournalChanges {
  const changes: JournalChanges = {}
  if (current.title.trim() !== saved.title) changes.title = current.title.trim()
  const theme: JournalChanges['theme'] = {}
  if (current.paper !== saved.paper) theme.paper = current.paper
  if (current.font !== saved.font) theme.font = current.font
  if (current.accent !== saved.accent) theme.accent = current.accent
  if (Object.keys(theme).length > 0) changes.theme = theme
  return changes
}

export default function Settings() {
  const { journal, setJournal } = useOutletContext<JournalContext>()
  if (!journal) {
    return (
      <p role="status" className="text-sm text-stone-500">
        Loading your journal…
      </p>
    )
  }
  // Keyed by id so the form starts from the loaded journal.
  return <SettingsForm key={journal.id} journal={journal} onSaved={setJournal} />
}

function SettingsForm({ journal, onSaved }: { journal: Journal; onSaved: (journal: Journal) => void }) {
  const navigate = useNavigate()
  const location = useLocation()
  const [form, setForm] = useState<Form>(() => formOf(journal))
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    setError('')
    setNotice('')
    const changes = changesBetween(formOf(journal), form)
    if (Object.keys(changes).length === 0) {
      setNotice('Nothing to save.')
      return
    }
    setBusy(true)
    try {
      const updated = await updateJournal(changes)
      onSaved(updated) // the frame restyles the whole journal at once
      setForm(formOf(updated))
      setNotice('Saved.')
    } catch (err) {
      if (err instanceof ApiError && err.status === 401) {
        navigate('/signin', { replace: true, state: { from: location.pathname } })
        return
      }
      if (err instanceof ApiError && err.status === 400) {
        setError(err.message)
      } else {
        const ref = err instanceof ApiError && err.requestId ? ` (reference: ${err.requestId})` : ''
        setError(`We couldn't save your settings. Please try again.${ref}`)
      }
    } finally {
      setBusy(false)
    }
  }

  return (
    <div className="max-w-xl">
      <Link to="/" className="text-sm underline">
        ← Your journal
      </Link>
      <h1 className="mt-2 text-2xl font-semibold">Journal settings</h1>

      <form onSubmit={handleSubmit} className="mt-4 space-y-5">
        <label className="block">
          <span className="text-sm">Journal title</span>
          <input
            required
            maxLength={MAX_JOURNAL_TITLE}
            value={form.title}
            onChange={(e) => setForm((f) => ({ ...f, title: e.target.value }))}
            className={inputClass}
          />
        </label>

        <fieldset>
          <legend className="text-sm">Paper</legend>
          <div className="mt-1 flex flex-wrap gap-4">
            {PAPERS.map((paper) => (
              <label key={paper} className="flex items-center gap-2 text-sm">
                <input
                  type="radio"
                  name="paper"
                  value={paper}
                  checked={form.paper === paper}
                  onChange={() => setForm((f) => ({ ...f, paper }))}
                />
                {paper}
              </label>
            ))}
          </div>
        </fieldset>

        <label className="block">
          <span className="text-sm">Font</span>
          <select
            value={form.font}
            onChange={(e) => setForm((f) => ({ ...f, font: e.target.value as Font }))}
            className={inputClass}
          >
            {FONTS.map((font) => (
              <option key={font} value={font}>
                {font}
              </option>
            ))}
          </select>
        </label>

        <div className="flex items-center gap-3">
          <label className="flex items-center gap-3">
            <span className="text-sm">Accent colour</span>
            {/* A colour picker always produces lower-case #rrggbb, exactly what the API accepts. */}
            <input
              type="color"
              value={form.accent}
              onChange={(e) => setForm((f) => ({ ...f, accent: e.target.value }))}
              className="h-9 w-14 cursor-pointer rounded border border-stone-300"
            />
          </label>
          <code className="text-sm">{form.accent}</code>
        </div>

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

        <button type="submit" disabled={busy} className="rounded bg-stone-900 px-4 py-2 text-white disabled:opacity-50">
          {busy ? 'Saving…' : 'Save settings'}
        </button>
      </form>
    </div>
  )
}