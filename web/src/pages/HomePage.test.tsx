import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { HomePage } from './HomePage'

// Mock the API
vi.mock('../api', () => ({
  listPrompts: vi.fn(),
  listTests: vi.fn(),
  listBenchmarks: vi.fn(),
  createPrompt: vi.fn(),
}))

import { listPrompts, listTests, listBenchmarks, createPrompt } from '../api'

const mockPrompts = [
  {
    id: '1',
    name: 'greeting',
    description: 'A friendly greeting prompt',
    file_path: 'greeting.prompt',
    version: '1.0.2',
    created_at: '2024-01-15T00:00:00Z',
  },
  {
    id: '2',
    name: 'summarize',
    description: 'Summarizes long text into key points',
    file_path: 'summarize.prompt',
    version: '2.1.0',
    created_at: '2024-01-14T00:00:00Z',
  },
  {
    id: '3',
    name: 'code-review',
    description: 'Reviews code and suggests improvements',
    file_path: 'code-review.prompt',
    version: '1.0.0',
    created_at: '2024-01-10T00:00:00Z',
  },
]

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('HomePage', () => {
  beforeEach(() => {
    vi.mocked(listPrompts).mockResolvedValue(mockPrompts)
    vi.mocked(listTests).mockResolvedValue([
      { name: 'greeting-tests', file_path: 'tests/greeting.test.yaml', prompt: 'greeting', test_count: 3 },
    ])
    vi.mocked(listBenchmarks).mockResolvedValue([
      { name: 'greeting-bench', file_path: 'benchmarks/greeting.bench.yaml', prompt: 'greeting', models: ['gpt-4o'], runs_per_model: 5 },
    ])
  })

  it('renders the page title', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /prompts/i })).toBeInTheDocument()
    })
  })

  it('shows prompt count', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText(/3 prompts tracked/i)).toBeInTheDocument()
    })
  })

  it('renders prompt cards', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('greeting')).toBeInTheDocument()
      expect(screen.getByText('summarize')).toBeInTheDocument()
      expect(screen.getByText('code-review')).toBeInTheDocument()
    })
  })

  it('shows version badges', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('v1.0.2')).toBeInTheDocument()
      expect(screen.getByText('v2.1.0')).toBeInTheDocument()
      expect(screen.getByText('v1.0.0')).toBeInTheDocument()
    })
  })

  it('prompt cards link to detail pages', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      const greetingLink = screen.getByRole('link', { name: /greeting/i })
      expect(greetingLink).toHaveAttribute('href', '/prompt/greeting')
    })
  })

  it('shows loading state initially', () => {
    renderWithRouter(<HomePage />)
    expect(screen.getByText(/loading prompts/i)).toBeInTheDocument()
  })

  it('shows error state on API failure', async () => {
    vi.mocked(listPrompts).mockRejectedValue(new Error('Network error'))
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText(/failed to load prompts/i)).toBeInTheDocument()
    })
  })

  it('shows empty state when no prompts', async () => {
    vi.mocked(listPrompts).mockResolvedValue([])
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText(/no prompts tracked yet/i)).toBeInTheDocument()
    })
  })

  it('filters prompts by search query', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText('greeting')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/search prompts/i)
    await user.type(searchInput, 'greeting')

    expect(screen.getByText('greeting')).toBeInTheDocument()
    expect(screen.queryByText('summarize')).not.toBeInTheDocument()
    expect(screen.queryByText('code-review')).not.toBeInTheDocument()
  })

  it('shows no results message when search has no matches', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText('greeting')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/search prompts/i)
    await user.type(searchInput, 'nonexistent')

    expect(screen.getByText(/no prompts matching "nonexistent"/i)).toBeInTheDocument()
  })

  it('shows New Prompt button', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('+ New Prompt')).toBeInTheDocument()
    })
  })

  it('opens modal when New Prompt clicked', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText('+ New Prompt')).toBeInTheDocument()
    })

    await user.click(screen.getByText('+ New Prompt'))
    expect(screen.getByText('New Prompt', { selector: 'h2' })).toBeInTheDocument()
    expect(screen.getByPlaceholderText('my-prompt')).toBeInTheDocument()
  })

  it('submits new prompt form', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    vi.mocked(createPrompt).mockResolvedValue({
      id: '4',
      name: 'new-one',
      description: 'A new prompt',
      file_path: 'prompts/new-one.prompt',
      version: '1.0.0',
      created_at: '2024-01-16T00:00:00Z',
    })
    renderWithRouter(<HomePage />)

    await waitFor(() => {
      expect(screen.getByText('+ New Prompt')).toBeInTheDocument()
    })

    await user.click(screen.getByText('+ New Prompt'))
    await user.type(screen.getByPlaceholderText('my-prompt'), 'new-one')
    await user.type(screen.getByPlaceholderText('What does this prompt do?'), 'A new prompt')
    await user.click(screen.getByText('Create'))

    await waitFor(() => {
      expect(createPrompt).toHaveBeenCalledWith('new-one', 'A new prompt', undefined)
    })
  })
})
