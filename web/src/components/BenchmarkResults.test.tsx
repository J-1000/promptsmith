import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { BenchmarkResults, BenchmarkResult } from './BenchmarkResults'

const mockResult: BenchmarkResult = {
  suiteName: 'summarizer-benchmark',
  promptName: 'summarizer',
  version: '1.2.0',
  models: [
    {
      model: 'gpt-4o',
      runs: 10,
      latencyP50Ms: 1500,
      latencyP99Ms: 2800,
      latencyAvgMs: 1650,
      totalTokensAvg: 847,
      promptTokens: 200,
      outputTokensAvg: 647,
      costPerRequest: 0.0042,
      totalCost: 0.042,
      errors: 0,
      errorRate: 0,
    },
    {
      model: 'gpt-4o-mini',
      runs: 10,
      latencyP50Ms: 800,
      latencyP99Ms: 1200,
      latencyAvgMs: 850,
      totalTokensAvg: 812,
      promptTokens: 200,
      outputTokensAvg: 612,
      costPerRequest: 0.0005,
      totalCost: 0.005,
      errors: 0,
      errorRate: 0,
    },
    {
      model: 'claude-sonnet',
      runs: 10,
      latencyP50Ms: 1200,
      latencyP99Ms: 2100,
      latencyAvgMs: 1350,
      totalTokensAvg: 825,
      promptTokens: 200,
      outputTokensAvg: 625,
      costPerRequest: 0.0038,
      totalCost: 0.038,
      errors: 1,
      errorRate: 0.1,
    },
  ],
  durationMs: 45000,
  startedAt: '2025-02-04T10:00:00Z',
  completedAt: '2025-02-04T10:00:45Z',
}

describe('BenchmarkResults', () => {
  it('renders empty state when no results', () => {
    render(<BenchmarkResults results={null} />)
    expect(screen.getByText('No benchmark results yet')).toBeInTheDocument()
  })

  it('renders run button in empty state when callback provided', () => {
    const onRunBenchmark = vi.fn()
    render(<BenchmarkResults results={null} onRunBenchmark={onRunBenchmark} />)

    const button = screen.getByRole('button', { name: 'Run Benchmark' })
    expect(button).toBeInTheDocument()

    fireEvent.click(button)
    expect(onRunBenchmark).toHaveBeenCalledTimes(1)
  })

  it('shows running state', () => {
    render(<BenchmarkResults results={null} onRunBenchmark={() => {}} isRunning />)
    expect(screen.getByRole('button', { name: 'Running...' })).toBeDisabled()
  })

  it('renders results table with models', () => {
    render(<BenchmarkResults results={mockResult} />)

    expect(screen.getByText('gpt-4o')).toBeInTheDocument()
    expect(screen.getByText('gpt-4o-mini')).toBeInTheDocument()
    expect(screen.getByText('claude-sonnet')).toBeInTheDocument()
  })

  it('shows suite name and version', () => {
    render(<BenchmarkResults results={mockResult} />)

    expect(screen.getByText('summarizer-benchmark')).toBeInTheDocument()
    expect(screen.getByText('v1.2.0')).toBeInTheDocument()
  })

  it('displays duration', () => {
    render(<BenchmarkResults results={mockResult} />)
    expect(screen.getByText('45.0s')).toBeInTheDocument()
  })

  it('shows recommendation when best model is same for both', () => {
    render(<BenchmarkResults results={mockResult} />)
    // gpt-4o-mini is best for both latency and cost
    expect(screen.getByText('gpt-4o-mini (best latency & cost)')).toBeInTheDocument()
  })

  it('shows separate recommendations when different models are best', () => {
    const diffBestResult: BenchmarkResult = {
      ...mockResult,
      models: [
        { ...mockResult.models[0], latencyP50Ms: 500 }, // gpt-4o fastest
        { ...mockResult.models[1], costPerRequest: 0.0001 }, // gpt-4o-mini cheapest
      ],
    }
    render(<BenchmarkResults results={diffBestResult} />)
    expect(screen.getByText('gpt-4o for speed')).toBeInTheDocument()
    expect(screen.getByText('gpt-4o-mini for cost')).toBeInTheDocument()
  })

  it('shows error count for models with errors', () => {
    render(<BenchmarkResults results={mockResult} />)
    expect(screen.getByText('1 (10%)')).toBeInTheDocument()
  })

  it('formats latency correctly', () => {
    render(<BenchmarkResults results={mockResult} />)
    // gpt-4o-mini has 800ms latency which should display as "800ms"
    expect(screen.getByText('800ms')).toBeInTheDocument()
    // gpt-4o has 1500ms which should display as "1.5s"
    expect(screen.getByText('1.5s')).toBeInTheDocument()
  })

  it('renders rerun button with results', () => {
    const onRunBenchmark = vi.fn()
    render(<BenchmarkResults results={mockResult} onRunBenchmark={onRunBenchmark} />)

    const button = screen.getByRole('button', { name: 'Re-run' })
    expect(button).toBeInTheDocument()

    fireEvent.click(button)
    expect(onRunBenchmark).toHaveBeenCalledTimes(1)
  })

  it('shows badges for best latency and cost', () => {
    render(<BenchmarkResults results={mockResult} />)
    // gpt-4o-mini should have both badges (fastest and cheapest)
    expect(screen.getByTitle('Fastest')).toBeInTheDocument()
    expect(screen.getByTitle('Cheapest')).toBeInTheDocument()
  })

  it('handles single model case', () => {
    const singleModelResult: BenchmarkResult = {
      ...mockResult,
      models: [mockResult.models[0]],
    }
    render(<BenchmarkResults results={singleModelResult} />)

    expect(screen.getByText('gpt-4o')).toBeInTheDocument()
    // No recommendation should be shown for single model
    expect(screen.queryByText('★')).not.toBeInTheDocument()
  })

  it('handles all models with errors', () => {
    const allErrorsResult: BenchmarkResult = {
      ...mockResult,
      models: [
        {
          ...mockResult.models[0],
          errors: 10,
          errorRate: 1.0,
          latencyP50Ms: 0,
          costPerRequest: 0,
        },
      ],
    }
    render(<BenchmarkResults results={allErrorsResult} />)

    // Should show dashes for metrics
    const cells = screen.getAllByText('—')
    expect(cells.length).toBeGreaterThan(0)
  })
})
