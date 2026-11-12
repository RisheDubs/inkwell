import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import App from './App'

describe('App', () => {
  it('shows the app name as the main heading', () => {
    render(<App />)
    // Find it by role and name, like a screen reader would, not by CSS class.
    expect(screen.getByRole('heading', { level: 1, name: 'Inkwell' })).toBeInTheDocument()
  })
})