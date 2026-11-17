import { renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { describe, expect, it, vi } from 'vitest'
import { ImageResolverContext, useImageResolver, type ImageResolver } from './imageResolver'

describe('useImageResolver', () => {
  it('fails loudly when no page chose a resolver', () => {
    // React logs the thrown error too; silence it so the test output stays readable.
    const spy = vi.spyOn(console, 'error').mockImplementation(() => {})
    expect(() => renderHook(() => useImageResolver())).toThrow(/ADR-0009/)
    spy.mockRestore()
  })

  it('returns the resolver the page provided', () => {
    const resolve: ImageResolver = vi.fn(async () => 'https://s3.example/x')
    const wrapper = ({ children }: { children: ReactNode }) => (
      <ImageResolverContext value={resolve}>{children}</ImageResolverContext>
    )

    const { result } = renderHook(() => useImageResolver(), { wrapper })

    expect(result.current).toBe(resolve)
  })
})