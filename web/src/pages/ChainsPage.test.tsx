import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ChainsPage } from './ChainsPage'

vi.mock('../api', () => ({
  listChains: vi.fn(),
  createChain: vi.fn(),
}))

import { listChains, createChain } from '../api'

const mockChains = [
  {
    id: '1',
    name: 'summarize-translate',
    description: 'Summarize then translate to French',
    step_count: 2,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
  {
    id: '2',
    name: 'expand-refine',
    description: '',
    step_count: 3,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
]

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('ChainsPage', () => {
  beforeEach(() => {
    vi.mocked(listChains).mockResolvedValue(mockChains)
    vi.mocked(createChain).mockResolvedValue(mockChains[0])
  })

  it('renders the page title', async () => {
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /prompt chains/i })).toBeInTheDocument()
    })
  })

  it('shows chain count', async () => {
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByText(/2 chains configured/i)).toBeInTheDocument()
    })
  })

  it('renders chain cards', async () => {
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByText('summarize-translate')).toBeInTheDocument()
      expect(screen.getByText('expand-refine')).toBeInTheDocument()
    })
  })

  it('shows step count badges', async () => {
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByText('2 steps')).toBeInTheDocument()
      expect(screen.getByText('3 steps')).toBeInTheDocument()
    })
  })

  it('shows description', async () => {
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByText('Summarize then translate to French')).toBeInTheDocument()
    })
  })

  it('shows loading state', () => {
    renderWithRouter(<ChainsPage />)
    expect(screen.getByText(/loading chains/i)).toBeInTheDocument()
  })

  it('shows error state on API failure', async () => {
    vi.mocked(listChains).mockRejectedValue(new Error('Network error'))
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByText(/failed to load chains/i)).toBeInTheDocument()
    })
  })

  it('shows empty state when no chains', async () => {
    vi.mocked(listChains).mockResolvedValue([])
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByText(/no chains found/i)).toBeInTheDocument()
    })
  })

  it('shows create button', async () => {
    renderWithRouter(<ChainsPage />)
    await waitFor(() => {
      expect(screen.getByText('+ New Chain')).toBeInTheDocument()
    })
  })

  it('opens create modal on button click', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter(<ChainsPage />)

    await waitFor(() => {
      expect(screen.getByText('+ New Chain')).toBeInTheDocument()
    })

    await user.click(screen.getByText('+ New Chain'))
    expect(screen.getByText('Create Chain')).toBeInTheDocument()
    expect(screen.getByPlaceholderText(/e.g. summarize-translate/i)).toBeInTheDocument()
  })

  it('filters chains by search', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter(<ChainsPage />)

    await waitFor(() => {
      expect(screen.getByText('summarize-translate')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/search chains/i)
    await user.type(searchInput, 'expand')

    expect(screen.getByText('expand-refine')).toBeInTheDocument()
    expect(screen.queryByText('summarize-translate')).not.toBeInTheDocument()
  })
})
