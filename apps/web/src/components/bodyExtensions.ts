import StarterKit from '@tiptap/starter-kit'
import { MediaImage } from './mediaImage'

// The node types an entry body may contain. The editor and the read-only view both use this
// list, so the public page can show everything the editor writes, and nothing else.
export function bodyExtensions() {
  return [
    StarterKit.configure({
      heading: { levels: [2, 3] }, // the page title is the only <h1>
      link: false, // links come later, with a safe-URL check
    }),
    MediaImage, // images by media ID (step 15)
  ]
}