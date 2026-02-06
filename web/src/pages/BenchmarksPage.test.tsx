import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BenchmarksPage } from './BenchmarksPage'

vi.mock('../api', () => ({
  listBenchmarks: vi.fn(),
}))

import { listBenchmarks } from '../api'

const mockBenchmarks = [
  {
    name: 'greeting-bench',
    file_path: 'benchmarks/greeting.bench.yaml',
    prompt: 'greeting',
    description: 'Benchmark greeting prompt',
    models: ['gpt-4o', 'claude-sonnet'],
    runs_per_model: 10,
  },
  {
    name: 'summarize-bench',
    file_path: 'benchmarks/summarize.bench.yaml',
    prompt: 'summarize',
    models: ['gpt-4o-mini'],
    runs_per_model: 5,
  },
]

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('BenchmarksPage', () => {
  beforeEach(() => {
    vi.mocked(listBenchmarks).mockResolvedValue(mockBenchmarks)
  })

  it('renders the page title', async () => {
    renderWithRouter(<BenchmarksPage />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /benchmarks/i })).toBeInTheDocument()
    })
  })

  it('shows benchmark count', async () => {
    renderWithRouter(<BenchmarksPage />)
    await waitFor(() => {
      expect(screen.getByText(/2 benchmarks configured/i)).toBeInTheDocument()
    })
  })

  it('renders benchmark cards with model tags', async () => {
    renderWithRouter(<BenchmarksPage />)
    await waitFor(() => {
      expect(screen.getByText('greeting-bench')).toBeInTheDocument()
      expect(screen.getByText('gpt-4o')).toBeInTheDocument()
      expect(screen.getByText('claude-sonnet')).toBeInTheDocument()
    })
  })

  it('shows loading state', () => {
    renderWithRouter(<BenchmarksPage />)
    expect(screen.getByText(/loading benchmarks/i)).toBeInTheDocument()
  })

  it('shows error state on API failure', async () => {
    vi.mocked(listBenchmarks).mockRejectedValue(new Error('Network error'))
    renderWithRouter(<BenchmarksPage />)
    await waitFor(() => {
      expect(screen.getByText(/failed to load benchmarks/i)).toBeInTheDocument()
    })
  })

  it('shows empty state when no benchmarks', async () => {
    vi.mocked(listBenchmarks).mockResolvedValue([])
    renderWithRouter(<BenchmarksPage />)
    await waitFor(() => {
      expect(screen.getByText(/no benchmarks found/i)).toBeInTheDocument()
    })
  })
})
