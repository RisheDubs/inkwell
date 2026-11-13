import { describe, expect, it } from 'vitest'
import { DEFAULT_THEME, themeStyle } from './theme'

describe('themeStyle', () => {
  it('turns a stored theme into a paper class and CSS variables', () => {
    expect(themeStyle({ paper: 'grid', font: 'mono', accent: '#aa3366' })).toEqual({
      className: 'paper-grid',
      style: { '--ink-accent': '#aa3366', '--ink-font': "ui-monospace, 'Cascadia Code', Menlo, monospace" },
    })
  })

  it('uses the default theme while the journal is still loading', () => {
    const { className, style } = themeStyle(undefined)
    expect(className).toBe(`paper-${DEFAULT_THEME.paper}`)
    expect(style).toMatchObject({ '--ink-accent': DEFAULT_THEME.accent })
  })

  it('never lets an unexpected value into CSS', () => {
    const hostile = {
      paper: 'lined; background:url(https://tracker.example)',
      font: 'x;}body{display:none',
      accent: 'red; background:url(https://tracker.example/?leak)',
    } as never
    const { className, style } = themeStyle(hostile)

    expect(className).toBe(`paper-${DEFAULT_THEME.paper}`)
    expect(style).toMatchObject({ '--ink-accent': DEFAULT_THEME.accent })
    expect(JSON.stringify(style)).not.toContain('tracker.example')
  })

  it('rejects colours the API would reject, such as names or upper-case hex', () => {
    expect(themeStyle({ accent: 'red' }).style).toMatchObject({ '--ink-accent': DEFAULT_THEME.accent })
    expect(themeStyle({ accent: '#AA3366' }).style).toMatchObject({ '--ink-accent': DEFAULT_THEME.accent })
  })
})