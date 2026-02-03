import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { Layout } from './Layout'

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('Layout', () => {
  it('renders the logo', () => {
    renderWithRouter(<Layout />)
    expect(screen.getByText('PromptSmith')).toBeInTheDocument()
  })

  it('renders navigation link to Prompts', () => {
    renderWithRouter(<Layout />)
    expect(screen.getByRole('link', { name: 'Prompts' })).toBeInTheDocument()
  })

  it('logo links to home page', () => {
    renderWithRouter(<Layout />)
    const logoLink = screen.getByRole('link', { name: /promptsmith/i })
    expect(logoLink).toHaveAttribute('href', '/')
  })
})
