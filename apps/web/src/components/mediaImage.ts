import { Node, ReactNodeViewRenderer } from '@tiptap/react'
import MediaImageView from './MediaImageView'

// An image in an entry body. It stores only { mediaId, alt }, never a URL:
// - links from our API expire after 15 minutes, so a stored link would break;
// - a stored URL could point anywhere (a tracking pixel, or javascript:), so the address always
//   comes from our API instead.
export const MediaImage = Node.create({
  name: 'mediaImage',
  group: 'block',
  atom: true, // one unit: no cursor inside it, deleted as a whole
  draggable: true,

  addAttributes() {
    return {
      mediaId: {
        default: null,
        parseHTML: (element) => element.getAttribute('data-media-id'),
        renderHTML: (attributes) => ({ 'data-media-id': attributes.mediaId }),
      },
      alt: { default: '' },
    }
  },

  // Only our own images survive copy and paste. An <img> pasted from another site is dropped,
  // so no outside URL (or tracking pixel) can get into a journal.
  parseHTML() {
    return [{ tag: 'img[data-media-id]' }]
  },

  renderHTML({ HTMLAttributes }) {
    return ['img', HTMLAttributes]
  },

  addNodeView() {
    return ReactNodeViewRenderer(MediaImageView)
  },
})