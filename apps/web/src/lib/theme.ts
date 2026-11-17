import type { CSSProperties } from 'react'
import { ACCENT_PATTERN, FONTS, PAPERS, type Font, type Paper, type Theme } from './journal'

// The API's defaults for a new journal (journal.go), used until the real theme arrives.
export const DEFAULT_THEME: Theme = { paper: 'dotted', font: 'serif', accent: '#2f5d50' }

// System fonts only: no web-font downloads, so no third-party requests and nothing extra for the
// Content Security Policy to allow.
const FONT_STACKS: Record<Font, string> = {
  serif: "Georgia, 'Times New Roman', serif",
  sans: "system-ui, 'Segoe UI', Roboto, sans-serif",
  mono: "ui-monospace, 'Cascadia Code', Menlo, monospace",
  handwriting: "'Segoe Print', 'Bradley Hand', 'Comic Sans MS', cursive",
}

const isPaper = (value: unknown): value is Paper => (PAPERS as readonly unknown[]).includes(value)
const isFont = (value: unknown): value is Font => (FONTS as readonly unknown[]).includes(value)

// Turns a stored theme into a class and CSS variables for the journal's wrapper. Every value is
// checked again here, although the API already validated it: these strings end up inside CSS,
// so only known-good values may pass, and anything else falls back to the default.
export function themeStyle(theme?: Partial<Theme>): { className: string; style: CSSProperties } {
  const paper = isPaper(theme?.paper) ? theme.paper : DEFAULT_THEME.paper
  const font = isFont(theme?.font) ? theme.font : DEFAULT_THEME.font
  const accent =
    typeof theme?.accent === 'string' && ACCENT_PATTERN.test(theme.accent) ? theme.accent : DEFAULT_THEME.accent

  return {
    className: `paper-${paper}`,
    style: { '--ink-accent': accent, '--ink-font': FONT_STACKS[font] } as CSSProperties,
  }
}