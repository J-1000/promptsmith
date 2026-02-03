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
})
