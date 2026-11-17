import { EditorContent, useEditor, type JSONContent } from '@tiptap/react'
import { useState } from 'react'
import { bodyExtensions } from './bodyExtensions'

type Props = {
  label: string // accessible name of the entry text
  content: JSONContent // read once: give the component a new key to show a different entry
}

// An entry body for reading, not editing: no toolbar, no uploads, nothing editable.
// Same node types as the editor (bodyExtensions). The body is Tiptap JSON from the API, treated
// as untrusted: Tiptap rebuilds it from known node types only, and anything it doesn't know is
// refused instead of guessed at.
export default function ReadOnlyBody({ label, content }: Props) {
  // Our own state, so setting it while Tiptap checks the content is allowed.
  const [unreadable, setUnreadable] = useState(false)

  const editor = useEditor({
    extensions: bodyExtensions(),
    content,
    editable: false,
    enableContentCheck: true,
    onContentError: () => setUnreadable(true),
    editorProps: {
      attributes: { 'aria-label': label, 'aria-readonly': 'true', class: 'p-3' },
    },
  })

  if (unreadable) {
    return <p className="text-sm text-stone-500">This entry can’t be shown.</p>
  }
  if (!editor) return null
  return <EditorContent editor={editor} />
}