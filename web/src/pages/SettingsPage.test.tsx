import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SettingsPage } from './SettingsPage'

vi.mock('../api', () => ({
  getProject: vi.fn(),
}))

import { getProject } from '../api'

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('SettingsPage', () => {
  beforeEach(() => {
    vi.mocked(getProject).mockResolvedValue({ id: 'proj_123', name: 'my-project' })
  })

  it('renders the page title', async () => {
    renderWithRouter(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /settings/i })).toBeInTheDocument()
    })
  })

  it('shows project info', async () => {
    renderWithRouter(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('my-project')).toBeInTheDocument()
      expect(screen.getByText('proj_123')).toBeInTheDocument()
    })
  })

  it('shows LLM providers section', async () => {
    renderWithRouter(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('OpenAI')).toBeInTheDocument()
      expect(screen.getByText('Anthropic')).toBeInTheDocument()
      expect(screen.getByText('Google')).toBeInTheDocument()
    })
  })

  it('shows env var names', async () => {
    renderWithRouter(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText('OPENAI_API_KEY')).toBeInTheDocument()
      expect(screen.getByText('ANTHROPIC_API_KEY')).toBeInTheDocument()
      expect(screen.getByText('GOOGLE_API_KEY')).toBeInTheDocument()
    })
  })

  it('shows sync section with CLI commands', async () => {
    renderWithRouter(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/configure cloud sync/i)).toBeInTheDocument()
    })
  })

  it('shows about section', async () => {
    renderWithRouter(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/the github copilot for prompt engineering/i)).toBeInTheDocument()
    })
  })

  it('shows error when project fails to load', async () => {
    vi.mocked(getProject).mockRejectedValue(new Error('Connection refused'))
    renderWithRouter(<SettingsPage />)
    await waitFor(() => {
      expect(screen.getByText(/could not load project info/i)).toBeInTheDocument()
    })
  })
})
