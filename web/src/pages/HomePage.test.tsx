import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { HomePage } from './HomePage'

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('HomePage', () => {
  it('renders the page title', () => {
    renderWithRouter(<HomePage />)
    expect(screen.getByRole('heading', { name: /prompts/i })).toBeInTheDocument()
  })

  it('shows prompt count', () => {
    renderWithRouter(<HomePage />)
    expect(screen.getByText(/3 prompts tracked/i)).toBeInTheDocument()
  })

  it('renders prompt cards', () => {
    renderWithRouter(<HomePage />)
    expect(screen.getByText('greeting')).toBeInTheDocument()
    expect(screen.getByText('summarize')).toBeInTheDocument()
    expect(screen.getByText('code-review')).toBeInTheDocument()
  })

  it('shows version badges', () => {
    renderWithRouter(<HomePage />)
    expect(screen.getByText('v1.0.2')).toBeInTheDocument()
    expect(screen.getByText('v2.1.0')).toBeInTheDocument()
    expect(screen.getByText('v1.0.0')).toBeInTheDocument()
  })

  it('shows tags on prompts', () => {
    renderWithRouter(<HomePage />)
    const prodTags = screen.getAllByText('prod')
    expect(prodTags.length).toBeGreaterThan(0)
    expect(screen.getByText('staging')).toBeInTheDocument()
  })

  it('prompt cards link to detail pages', () => {
    renderWithRouter(<HomePage />)
    const greetingLink = screen.getByRole('link', { name: /greeting/i })
    expect(greetingLink).toHaveAttribute('href', '/prompt/greeting')
  })
})
