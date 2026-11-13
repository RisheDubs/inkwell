// Adds DOM matchers such as toBeInTheDocument() to Vitest's expect.
import '@testing-library/jest-dom/vitest'
import { afterEach } from 'vitest'
import { cleanup } from '@testing-library/react'

// Unmount whatever a test rendered, so tests can't leak into each other.
afterEach(() => cleanup())

// jsdom has no layout engine. ProseMirror (inside Tiptap) asks where things are on screen when
// it scrolls the cursor into view; answer "nowhere" so editor tests can run in Node.
const emptyRect = { x: 0, y: 0, width: 0, height: 0, top: 0, right: 0, bottom: 0, left: 0, toJSON: () => ({}) } as DOMRect
const noRects = (() => Object.assign([], { item: () => null })) as unknown as () => DOMRectList
Range.prototype.getClientRects ??= noRects
Range.prototype.getBoundingClientRect ??= () => emptyRect
Element.prototype.getClientRects ??= noRects
document.elementFromPoint ??= () => null