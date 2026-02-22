import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter, Routes, Route } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PromptPage } from './PromptPage'

// Mock the API
vi.mock('../api', () => ({
  getPrompt: vi.fn(),
  getPromptVersions: vi.fn(),
  getPromptDiff: vi.fn(),
  deletePrompt: vi.fn(),
  createTag: vi.fn(),
  deleteTag: vi.fn(),
  runTest: vi.fn(),
  runBenchmark: vi.fn(),
  generateVariations: vi.fn(),
  listTests: vi.fn(),
  listBenchmarks: vi.fn(),
}))

import {
  getPrompt,
  getPromptVersions,
  getPromptDiff,
  deletePrompt,
  runTest,
  runBenchmark,
  listTests,
  listBenchmarks,
} from '../api'

const mockPrompt = {
  id: '1',
  name: 'greeting',
  description: 'A friendly greeting prompt',
  file_path: 'greeting.prompt',
  version: '1.0.2',
  created_at: '2024-01-15T00:00:00Z',
}

const mockVersions = [
  {
    id: '1',
    version: '1.0.2',
    content: 'You are a helpful assistant. Greet the user {{user_name}} in a {{tone}} manner.',
    commit_message: 'Add tone parameter for flexibility',
    created_at: '2024-01-15T14:32:00Z',
    tags: ['prod'],
  },
  {
    id: '2',
    version: '1.0.1',
    content: 'You are a helpful assistant. Greet the user {{user_name}}.',
    commit_message: 'Fix greeting for edge cases',
    created_at: '2024-01-12T09:15:00Z',
    tags: [],
  },
  {
    id: '3',
    version: '1.0.0',
    content: 'Greet the user.',
    commit_message: 'Initial version of greeting prompt',
    created_at: '2024-01-10T16:45:00Z',
    tags: ['staging'],
  },
]

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
  beforeEach(() => {
    vi.mocked(getPrompt).mockResolvedValue(mockPrompt)
    vi.mocked(getPromptVersions).mockResolvedValue(mockVersions)
    vi.mocked(getPromptDiff).mockResolvedValue({
      prompt: 'greeting',
      v1: { version: '1.0.1', content: mockVersions[1].content },
      v2: { version: '1.0.2', content: mockVersions[0].content },
    })
    vi.mocked(listTests).mockResolvedValue([])
    vi.mocked(listBenchmarks).mockResolvedValue([])
  })

  it('renders prompt name from URL', async () => {
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /greeting/i })).toBeInTheDocument()
    })
  })

  it('renders breadcrumb navigation', async () => {
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByText('Prompts')).toBeInTheDocument()
    })
  })

  it('shows current version', async () => {
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      // Version appears in header badge
      const versionElements = screen.getAllByText('v1.0.2')
      expect(versionElements.length).toBeGreaterThan(0)
    })
  })

  it('renders tab buttons', async () => {
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /content/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /history/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /diff/i })).toBeInTheDocument()
    })
  })

  it('shows content view by default', async () => {
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByText('greeting.prompt')).toBeInTheDocument()
      expect(screen.getByText(/You are a helpful assistant/)).toBeInTheDocument()
    })
  })

  it('switches to history view when tab clicked', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /history/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /history/i }))

    expect(screen.getByText(/Select two versions to compare/i)).toBeInTheDocument()
    expect(screen.getByText('v1.0.1')).toBeInTheDocument()
  })

  it('enables diff tab after selecting two versions', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /history/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /history/i }))

    // Select first version
    await user.click(screen.getByText('Add tone parameter for flexibility'))
    // Select second version
    await user.click(screen.getByText('Fix greeting for edge cases'))

    const diffTab = screen.getByRole('button', { name: /diff/i })
    expect(diffTab).not.toBeDisabled()
  })

  it('shows loading state initially', () => {
    renderWithRouter('/prompt/greeting')
    expect(screen.getByText(/loading prompt/i)).toBeInTheDocument()
  })

  it('shows error state on API failure', async () => {
    vi.mocked(getPrompt).mockRejectedValue(new Error('Not found'))
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByText(/failed to load prompt/i)).toBeInTheDocument()
    })
  })

  it('renders generate tab', async () => {
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /generate/i })).toBeInTheDocument()
    })
  })

  it('switches to generate view when tab clicked', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /generate/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /generate/i }))

    expect(screen.getByText('Generate variations of your prompt using AI')).toBeInTheDocument()
  })

  it('shows delete button', async () => {
    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByText('Delete')).toBeInTheDocument()
    })
  })

  it('shows confirm dialog on delete click', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByText('Delete')).toBeInTheDocument()
    })

    await user.click(screen.getByText('Delete'))
    expect(screen.getByText('Delete Prompt')).toBeInTheDocument()
    expect(screen.getByText(/are you sure/i)).toBeInTheDocument()
  })

  it('calls deletePrompt on confirm', async () => {
    const user = userEvent.setup()
    vi.mocked(deletePrompt).mockResolvedValue(undefined)
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByText('Delete')).toBeInTheDocument()
    })

    await user.click(screen.getByText('Delete'))
    // Click confirm in dialog — the confirm button has label "Delete"
    const dialogButtons = screen.getAllByText('Delete')
    await user.click(dialogButtons[dialogButtons.length - 1])

    await waitFor(() => {
      expect(deletePrompt).toHaveBeenCalledWith('greeting')
    })
  })

  it('shows add tag button in history view', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /history/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /history/i }))

    const addTagButtons = screen.getAllByText('+ tag')
    expect(addTagButtons.length).toBeGreaterThan(0)
  })

  it('shows change impact in diff view', async () => {
    vi.mocked(listTests).mockResolvedValue([
      { name: 'greeting-tests', file_path: 'tests/greeting.test.yaml', prompt: 'greeting', test_count: 5 },
    ])
    vi.mocked(listBenchmarks).mockResolvedValue([
      { name: 'greeting-bench', file_path: 'benchmarks/greeting.bench.yaml', prompt: 'greeting', models: [], runs_per_model: 10 },
    ])
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /history/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /history/i }))
    await user.click(screen.getByText('Add tone parameter for flexibility'))
    await user.click(screen.getByText('Fix greeting for edge cases'))
    await user.click(screen.getByRole('button', { name: /diff/i }))

    await waitFor(() => {
      expect(screen.getByText('Change Impact')).toBeInTheDocument()
      expect(screen.getByText('greeting-tests')).toBeInTheDocument()
      expect(screen.getByText('greeting-bench')).toBeInTheDocument()
    })
  })

  it('runs tests using suite name mapped to prompt', async () => {
    const user = userEvent.setup()
    vi.mocked(listTests).mockResolvedValue([
      { name: 'greeting-suite', file_path: 'tests/greeting.test.yaml', prompt: 'greeting', test_count: 2 },
    ])
    vi.mocked(runTest).mockResolvedValue({
      suite_name: 'greeting-suite',
      prompt_name: 'greeting',
      version: '1.0.2',
      passed: 2,
      failed: 0,
      skipped: 0,
      total: 2,
      results: [],
      duration_ms: 123,
    })

    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /tests/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /^Tests/i }))
    await user.click(screen.getByRole('button', { name: 'Run Tests' }))

    await waitFor(() => {
      expect(runTest).toHaveBeenCalledWith('greeting-suite')
    })
  })

  it('runs benchmark using suite name mapped to prompt', async () => {
    const user = userEvent.setup()
    vi.mocked(listBenchmarks).mockResolvedValue([
      {
        name: 'greeting-bench',
        file_path: 'benchmarks/greeting.bench.yaml',
        prompt: 'greeting',
        models: ['gpt-4o-mini'],
        runs_per_model: 2,
      },
    ])
    vi.mocked(runBenchmark).mockResolvedValue({
      suite_name: 'greeting-bench',
      prompt_name: 'greeting',
      version: '1.0.2',
      models: [
        {
          model: 'gpt-4o-mini',
          runs: 2,
          errors: 0,
          error_rate: 0,
          latency_p50_ms: 100,
          latency_p99_ms: 120,
          total_tokens_avg: 50,
          cost_per_request: 0.001,
        },
      ],
      duration_ms: 456,
    })

    renderWithRouter('/prompt/greeting')
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /benchmarks/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /^Benchmarks/i }))
    await user.click(screen.getByRole('button', { name: 'Run Benchmark' }))

    await waitFor(() => {
      expect(runBenchmark).toHaveBeenCalledWith('greeting-bench')
    })
  })

  it('shows remove tag buttons in history view', async () => {
    const user = userEvent.setup()
    renderWithRouter('/prompt/greeting')

    await waitFor(() => {
      expect(screen.getByRole('button', { name: /history/i })).toBeInTheDocument()
    })

    await user.click(screen.getByRole('button', { name: /history/i }))

    expect(screen.getByLabelText('Remove tag prod')).toBeInTheDocument()
    expect(screen.getByLabelText('Remove tag staging')).toBeInTheDocument()
  })
})
