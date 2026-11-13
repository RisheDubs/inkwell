import {
  EditorContent,
  useEditor,
  useEditorState,
  type Editor as TiptapEditor,
  type JSONContent,
} from '@tiptap/react'
import StarterKit from '@tiptap/starter-kit'
import type { TiptapDoc } from '../lib/entries'

const EMPTY_DOC: TiptapDoc = { type: 'doc', content: [{ type: 'paragraph' }] }

type Props = {
  label: string // accessible name of the writing area
  initialContent?: TiptapDoc // read once, when the editor is created
  onChange: (doc: TiptapDoc) => void
  // The stored body has content this editor can't show (an unknown node type). The editor
  // locks itself so a save can never overwrite the real body with an empty one.
  onContentError?: () => void
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
    </div>
  )
}

// Rich-text editor. Stores and loads Tiptap JSON, never HTML: Tiptap rebuilds the document from
// the node types it knows, so stored text can't become a <script> or an onerror= attribute.
export default function Editor({ label, initialContent, onChange, onContentError }: Props) {
  const editor = useEditor({
    extensions: [
      StarterKit.configure({
        heading: { levels: [2, 3] }, // the page title is the only <h1>
        link: false, // links come later, with a safe-URL check
      }),
    ],
    content: (initialContent ?? EMPTY_DOC) as JSONContent,
    enableContentCheck: true, // report unknown content instead of silently dropping it
    onContentError: ({ editor: e }) => {
      e.setEditable(false)
      onContentError?.()
    },
    editorProps: {
      attributes: { role: 'textbox', 'aria-multiline': 'true', 'aria-label': label, class: 'p-3' },
    },
    onUpdate: ({ editor: e }) => onChange(e.getJSON() as TiptapDoc),
  })

  if (!editor) return null

  return (
    <div className="rounded border border-stone-300 bg-white">
      <Toolbar editor={editor} />
      <EditorContent editor={editor} />
    </div>
  )
}