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
  getDashboardActivity: vi.fn(),
  getDashboardHealth: vi.fn(),
}))

import { listPrompts, listTests, listBenchmarks, createPrompt, getDashboardActivity, getDashboardHealth } from '../api'

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

const mockActivity = [
  {
    type: 'version',
    title: 'v1.0.2',
    detail: 'Added error handling',
    timestamp: new Date().toISOString(),
    prompt_name: 'greeting',
  },
  {
    type: 'test_run',
    title: 'passed',
    detail: 'greeting-tests',
    timestamp: new Date(Date.now() - 3600000).toISOString(),
    prompt_name: 'greeting',
  },
  {
    type: 'benchmark_run',
    title: 'completed',
    detail: 'greeting-bench',
    timestamp: new Date(Date.now() - 7200000).toISOString(),
    prompt_name: 'greeting',
  },
]

const mockHealth = [
  {
    prompt_name: 'greeting',
    version_count: 3,
    last_test_status: 'passed',
    last_test_at: '2024-01-15T00:00:00Z',
    test_pass_rate: 1.0,
  },
  {
    prompt_name: 'summarize',
    version_count: 5,
    last_test_status: 'failed',
    last_test_at: '2024-01-14T00:00:00Z',
    test_pass_rate: 0.67,
  },
  {
    prompt_name: 'code-review',
    version_count: 1,
    last_test_status: 'none',
    last_test_at: '',
    test_pass_rate: 0,
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
    vi.mocked(getDashboardActivity).mockResolvedValue(mockActivity)
    vi.mocked(getDashboardHealth).mockResolvedValue(mockHealth)
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
      // Use getAllByText since 'greeting' appears in both card and activity
      expect(screen.getAllByText('greeting').length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByText('summarize').length).toBeGreaterThanOrEqual(1)
      expect(screen.getAllByText('code-review').length).toBeGreaterThanOrEqual(1)
    })
  })

  it('shows version badges', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      // v1.0.2 appears in both version badge and activity event
      expect(screen.getAllByText('v1.0.2').length).toBeGreaterThanOrEqual(1)
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
      expect(screen.getAllByText('greeting').length).toBeGreaterThanOrEqual(1)
    })

    const searchInput = screen.getByPlaceholderText(/search prompts/i)
    await user.type(searchInput, 'summarize')

    // After filtering, only summarize card should be in the grid
    expect(screen.getAllByText('summarize').length).toBeGreaterThanOrEqual(1)
    // greeting still appears in activity feed but not in filtered cards
    expect(screen.queryByText('code-review')).not.toBeInTheDocument()
  })

  it('shows no results message when search has no matches', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter(<HomePage />)

    await waitFor(() => {
      expect(screen.getAllByText('greeting').length).toBeGreaterThanOrEqual(1)
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

  // Activity feed tests

  it('renders activity feed section', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('Recent Activity')).toBeInTheDocument()
    })
  })

  it('renders activity events with details', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('Added error handling')).toBeInTheDocument()
      expect(screen.getByText('greeting-tests')).toBeInTheDocument()
      expect(screen.getByText('greeting-bench')).toBeInTheDocument()
    })
  })

  it('shows prompt names in activity events', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      // 'greeting' appears in card + multiple activity badges
      const promptBadges = screen.getAllByText('greeting')
      expect(promptBadges.length).toBeGreaterThanOrEqual(2)
    })
  })

  it('hides activity section when no activity', async () => {
    vi.mocked(getDashboardActivity).mockResolvedValue([])
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /prompts/i })).toBeInTheDocument()
    })
    expect(screen.queryByText('Recent Activity')).not.toBeInTheDocument()
  })

  // Health indicator tests

  it('shows health dot for prompts with passing tests', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('100% passing')).toBeInTheDocument()
    })
  })

  it('shows pass rate for prompts with failing tests', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getByText('67% passing')).toBeInTheDocument()
    })
  })

  it('does not show pass rate for prompts with no tests', async () => {
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      expect(screen.getAllByText('code-review').length).toBeGreaterThanOrEqual(1)
    })
    expect(screen.queryByText('0% passing')).not.toBeInTheDocument()
  })

  it('handles health API failure gracefully', async () => {
    vi.mocked(getDashboardHealth).mockRejectedValue(new Error('Failed'))
    renderWithRouter(<HomePage />)
    await waitFor(() => {
      // Cards still render despite health API failure
      expect(screen.getAllByText('greeting').length).toBeGreaterThanOrEqual(1)
    })
    expect(screen.queryByText('100% passing')).not.toBeInTheDocument()
  })
})
