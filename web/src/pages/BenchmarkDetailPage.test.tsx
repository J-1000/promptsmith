import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BenchmarkDetailPage } from './BenchmarkDetailPage'

vi.mock('../api', () => ({
  getBenchmark: vi.fn(),
  runBenchmark: vi.fn(),
}))

import { getBenchmark } from '../api'

const mockBenchmark = {
  name: 'greeting-bench',
  file_path: 'benchmarks/greeting.bench.yaml',
  prompt: 'greeting',
  description: 'Benchmark the greeting prompt',
  models: ['gpt-4o', 'claude-sonnet'],
  runs_per_model: 10,
}

function renderWithRoute() {
  return render(
    <MemoryRouter initialEntries={['/benchmarks/greeting-bench']}>
      <Routes>
        <Route path="/benchmarks/:name" element={<BenchmarkDetailPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('BenchmarkDetailPage', () => {
  beforeEach(() => {
    vi.mocked(getBenchmark).mockResolvedValue(mockBenchmark)
  })

  it('shows loading state initially', () => {
    renderWithRoute()
    expect(screen.getByText(/loading benchmark/i)).toBeInTheDocument()
  })

  it('renders benchmark name', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /greeting-bench/i })).toBeInTheDocument()
    })
  })

  it('shows model tags', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('gpt-4o')).toBeInTheDocument()
      expect(screen.getByText('claude-sonnet')).toBeInTheDocument()
    })
  })

  it('shows runs per model', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('10 runs/model')).toBeInTheDocument()
    })
  })

  it('shows link to associated prompt', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('greeting')).toBeInTheDocument()
    })
  })

  it('shows run benchmark button', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /run benchmark/i })).toBeInTheDocument()
    })
  })

  it('shows error state on API failure', async () => {
    vi.mocked(getBenchmark).mockRejectedValue(new Error('Not found'))
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText(/failed to load benchmark/i)).toBeInTheDocument()
    })
  })
})
