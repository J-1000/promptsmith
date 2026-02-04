import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { describe, it, expect } from 'vitest'
import { PromptPage } from './PromptPage'

function renderWithRouter(initialRoute = '/prompt/greeting') {
  window.history.pushState({}, '', initialRoute)
  return render(
    <BrowserRouter>
      <Routes>
        <Route path="/prompt/:name" element={<PromptPage />} />
      </Routes>
    </BrowserRouter>
  )
}

describe('PromptPage', () => {
  it('renders prompt name from URL', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByRole('heading', { name: /greeting/i })).toBeInTheDocument()
  })

  it('renders breadcrumb navigation', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByText('Prompts')).toBeInTheDocument()
  })

  it('shows current version', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByText('v1.0.2')).toBeInTheDocument()
  })

  it('renders tab buttons', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByRole('button', { name: /content/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /history/i })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /diff/i })).toBeInTheDocument()
  })

  it('shows content view by default', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByText('greeting.prompt')).toBeInTheDocument()
    expect(screen.getByText(/You are a helpful assistant/)).toBeInTheDocument()
  })

  it('switches to history view when tab clicked', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await user.click(screen.getByRole('button', { name: /history/i }))

    expect(screen.getByText(/Select two versions to compare/i)).toBeInTheDocument()
    expect(screen.getByText('v1.0.1')).toBeInTheDocument()
  })

  it('enables diff tab after selecting two versions', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await user.click(screen.getByRole('button', { name: /history/i }))

    // Select first version
    await user.click(screen.getByText('Add tone parameter for flexibility'))
    // Select second version
    await user.click(screen.getByText('Fix greeting for edge cases'))

    const diffTab = screen.getByRole('button', { name: /diff/i })
    expect(diffTab).not.toBeDisabled()
  })

  it('renders tests tab with results count', () => {
    renderWithRouter('/prompt/greeting')
    // Tests tab should show pass/total count
    expect(screen.getByRole('button', { name: /tests/i })).toBeInTheDocument()
    expect(screen.getByText('3/5')).toBeInTheDocument()
  })

  it('switches to tests view when tab clicked', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await user.click(screen.getByRole('button', { name: /tests/i }))

    // Should show test results
    expect(screen.getByText('3 passed')).toBeInTheDocument()
    expect(screen.getByText('1 failed')).toBeInTheDocument()
  })

  it('renders benchmarks tab with model count', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByRole('button', { name: /benchmarks/i })).toBeInTheDocument()
    // Should show model count badge
    expect(screen.getByText('3')).toBeInTheDocument()
  })

  it('switches to benchmarks view when tab clicked', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await user.click(screen.getByRole('button', { name: /benchmarks/i }))

    // Should show benchmark results
    expect(screen.getByText('gpt-4o')).toBeInTheDocument()
    expect(screen.getByText('gpt-4o-mini')).toBeInTheDocument()
    expect(screen.getByText('claude-sonnet')).toBeInTheDocument()
  })

  it('shows recommendation in benchmarks view', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await user.click(screen.getByRole('button', { name: /benchmarks/i }))

    // gpt-4o-mini should be recommended (fastest and cheapest in mock data)
    expect(screen.getByText('gpt-4o-mini (best latency & cost)')).toBeInTheDocument()
  })

  it('renders generate tab', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByRole('button', { name: /generate/i })).toBeInTheDocument()
  })

  it('switches to generate view when tab clicked', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await user.click(screen.getByRole('button', { name: /generate/i }))

    // Should show generate controls
    expect(screen.getByText('Generate variations of your prompt using AI')).toBeInTheDocument()
  })

  it('shows generation type buttons in generate view', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await user.click(screen.getByRole('button', { name: /^generate$/i }))

    // Should show type buttons
    expect(screen.getByRole('button', { name: 'variations' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'compress' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'expand' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'rephrase' })).toBeInTheDocument()
  })
})
