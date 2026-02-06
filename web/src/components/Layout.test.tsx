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

  it('renders navigation links', () => {
    renderWithRouter(<Layout />)
    expect(screen.getByRole('link', { name: 'Prompts' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Tests' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Benchmarks' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Settings' })).toBeInTheDocument()
  })

  it('logo links to home page', () => {
    renderWithRouter(<Layout />)
    const logoLink = screen.getByRole('link', { name: /promptsmith/i })
    expect(logoLink).toHaveAttribute('href', '/')
  })

  it('nav links point to correct routes', () => {
    renderWithRouter(<Layout />)
    expect(screen.getByRole('link', { name: 'Tests' })).toHaveAttribute('href', '/tests')
    expect(screen.getByRole('link', { name: 'Benchmarks' })).toHaveAttribute('href', '/benchmarks')
    expect(screen.getByRole('link', { name: 'Settings' })).toHaveAttribute('href', '/settings')
  })
})
