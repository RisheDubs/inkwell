import { describe, expect, it } from 'vitest'
import { handleProblem, normalizeHandle, publicPath, publicUrl } from './handle'

describe('handles', () => {
  it('normalises what the user typed the way the API stores it', () => {
    expect(normalizeHandle('  Rishekesh ')).toBe('rishekesh')
  })

  it.each(['rishekesh', 'abc', 'a_b-c', '0day', 'a'.repeat(30)])('accepts %s', (h) => {
    expect(handleProblem(h)).toBeUndefined()
  })

  it.each(['', 'ab', '-abc', '_abc', 'Rishi', 'has space', 'a/b', '../me', 'a'.repeat(31)])('rejects %j', (h) => {
    expect(handleProblem(h)).toMatch(/3–30 characters/)
  })

  it('builds the public path and the full link to share', () => {
    expect(publicPath('rishekesh')).toBe('/j/rishekesh')
    expect(publicUrl('rishekesh', 'http://localhost:5173')).toBe('http://localhost:5173/j/rishekesh')
  })
})