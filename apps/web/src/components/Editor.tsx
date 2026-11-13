import { useEffect, useRef, useState, type ChangeEvent } from 'react'
import {
  EditorContent,
  useEditor,
  useEditorState,
  type Editor as TiptapEditor,
  type JSONContent,
} from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import type { TiptapDoc } from '../lib/entries'
import { IMAGE_TYPES, UploadError, uploadImage } from '../lib/media'
import { MediaImage } from './mediaImage'

const EMPTY_DOC: TiptapDoc = { type: 'doc', content: [{ type: 'paragraph' }] }

type Props = {
  label: string // accessible name of the writing area
  initialContent?: TiptapDoc // read once, when the editor is created
  onChange: (doc: TiptapDoc) => void
  // The stored body has content this editor can't show (an unknown node type). The editor
  // locks itself so a save can never overwrite the real body with an empty one.
  onContentError?: () => void
}

// Toolbar button that uploads an image straight to S3, then inserts it at the cursor.
function ImageButton({ editor, disabled }: { editor: TiptapEditor; disabled: boolean }) {
  const input = useRef<HTMLInputElement>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  async function handleFile(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    event.target.value = '' // so choosing the same file again still fires a change
    if (!file) return
    setError('')
    setBusy(true)
    try {
      const mediaId = await uploadImage(file)
      editor.chain().focus().insertContent({ type: 'mediaImage', attrs: { mediaId } }).run()
    } catch (err) {
      setError(err instanceof UploadError ? err.message : 'The upload failed. Please try again.')
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <button
        type="button"
        disabled={disabled || busy}
        onClick={() => input.current?.click()}
        className="rounded px-2 py-1 text-sm hover:bg-stone-100 disabled:opacity-50"
      >
        {busy ? 'Uploading…' : 'Image'}
      </button>
      <input
        ref={input}
        type="file"
        accept={IMAGE_TYPES.join(',')}
        hidden
        aria-label="Choose an image"
        onChange={(e) => void handleFile(e)}
      />
      {error && (
        <p role="alert" className="w-full px-2 text-sm text-red-700">
          {error}
        </p>
      )}
    </>
  )
}

function Toolbar({ editor }: { editor: TiptapEditor }) {
  // Re-render the toolbar only when these values change, not on every keystroke.
  const active = useEditorState({
    editor,
    selector: ({ editor: e }) => ({
      bold: e.isActive('bold'),
      italic: e.isActive('italic'),
      heading: e.isActive('heading', { level: 2 }),
      bulletList: e.isActive('bulletList'),
      orderedList: e.isActive('orderedList'),
      editable: e.isEditable,
    }),
  })

  const buttons = [
    { label: 'Bold', pressed: active.bold, run: () => editor.chain().focus().toggleBold().run() },
    { label: 'Italic', pressed: active.italic, run: () => editor.chain().focus().toggleItalic().run() },
    { label: 'Heading', pressed: active.heading, run: () => editor.chain().focus().toggleHeading({ level: 2 }).run() },
    { label: 'Bullet list', pressed: active.bulletList, run: () => editor.chain().focus().toggleBulletList().run() },
    { label: 'Numbered list', pressed: active.orderedList, run: () => editor.chain().focus().toggleOrderedList().run() },
  ]

  return (
    <div role="toolbar" aria-label="Formatting" className="flex flex-wrap gap-1 border-b border-stone-200 p-1">
      {buttons.map((b) => (
        <button
          key={b.label}
          type="button"
          aria-pressed={b.pressed}
          disabled={!active.editable}
          onClick={b.run}
          className="rounded px-2 py-1 text-sm hover:bg-stone-100 disabled:opacity-50 aria-pressed:bg-stone-200"
        >
          {b.label}
        </button>
      ))}
      <ImageButton editor={editor} disabled={!active.editable} />
    </div>
  )
}

// Rich-text editor. Stores and loads Tiptap JSON, never HTML: Tiptap rebuilds the document from
// the node types it knows, so stored text can't become a <script> or an onerror= attribute.
export default function Editor({ label, initialContent, onChange, onContentError }: Props) {
  // Tiptap checks the content while this component is still rendering. Telling the parent right
  // then would update the parent mid-render, which React forbids; so remember it here and tell
  // the parent in an effect, after the render has finished.
  const [unreadable, setUnreadable] = useState(false)

  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [2, 3] }, // the page title is the only <h1>
        link: false, // links come later, with a safe-URL check
      }),
      MediaImage, // images by media ID (step 15)
    ],
    content: (initialContent ?? EMPTY_DOC) as JSONContent,
    enableContentCheck: true, // report unknown content instead of silently dropping it
    onContentError: ({ editor: e }) => {
      e.setEditable(false)
      setUnreadable(true)
    },
    editorProps: {
      attributes: { role: 'textbox', 'aria-multiline': 'true', 'aria-label': label, class: 'p-3' },
    },
    // Report only real edits. Tiptap also fires an update while loading content it can't show,
    // carrying an empty document; passing that on would look like the user deleted their text.
    onUpdate: ({ editor: e, transaction }) => {
      if (transaction.docChanged && e.isEditable) onChange(e.getJSON() as TiptapDoc)
    },
  })

  useEffect(() => {
    if (unreadable) onContentError?.()
  }, [unreadable, onContentError])

  if (!editor) return null

  return (
    <div className="rounded border border-stone-300 bg-white">
      <Toolbar editor={editor} />
      <EditorContent editor={editor} />
    </div>
  )
}