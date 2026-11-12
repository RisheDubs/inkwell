import { render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router'
import { describe, expect, it } from 'vitest'
import { routes } from './router'

// A memory router keeps the URL in memory, so tests can start on any path without a browser.
function renderAt(path: string) {
  render(<RouterProvider router={createMemoryRouter(routes, { initialEntries: [path] })} />)
}

describe('routes', () => {
  it.each([
    ['/', 'Your journal'],
    ['/signin', 'Sign in'],
    ['/signup', 'Create your account'],
    ['/no-such-page', 'Page not found'],
  ])('%s shows "%s"', (path, heading) => {
    renderAt(path)
    expect(screen.getByRole('heading', { level: 1, name: heading })).toBeInTheDocument()
  })
})