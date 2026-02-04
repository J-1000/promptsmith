import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TestResults, SuiteResult } from './TestResults'

const mockResults: SuiteResult = {
  suiteName: 'test-suite',
  promptName: 'greeting',
  version: '1.0.0',
  passed: 2,
  failed: 1,
  skipped: 1,
  total: 4,
  durationMs: 50,
  results: [
    {
      testName: 'passing-test',
      passed: true,
      skipped: false,
      durationMs: 10,
    },
    {
      testName: 'another-passing',
      passed: true,
      skipped: false,
      durationMs: 12,
    },
    {
      testName: 'failing-test',
      passed: false,
      skipped: false,
      durationMs: 15,
      failures: [
        {
          type: 'contains',
          message: 'expected output to contain "hello"',
          expected: 'hello',
          actual: 'goodbye',
        },
      ],
    },
    {
      testName: 'skipped-test',
      passed: false,
      skipped: true,
      durationMs: 0,
    },
  ],
}

describe('TestResults', () => {
  it('renders empty state when no results', () => {
    render(<TestResults results={null} />)
    expect(screen.getByText('No test results yet')).toBeInTheDocument()
  })

  it('renders run button in empty state', () => {
    const onRunTests = vi.fn()
    render(<TestResults results={null} onRunTests={onRunTests} />)

    const button = screen.getByRole('button', { name: /run tests/i })
    expect(button).toBeInTheDocument()

    fireEvent.click(button)
    expect(onRunTests).toHaveBeenCalled()
  })

  it('disables run button when running', () => {
    const onRunTests = vi.fn()
    render(<TestResults results={null} onRunTests={onRunTests} isRunning={true} />)

    const button = screen.getByRole('button', { name: /running/i })
    expect(button).toBeDisabled()
  })

  it('renders summary stats', () => {
    render(<TestResults results={mockResults} />)

    expect(screen.getByText('2 passed')).toBeInTheDocument()
    expect(screen.getByText('1 failed')).toBeInTheDocument()
    expect(screen.getByText('1 skipped')).toBeInTheDocument()
    expect(screen.getByText('4 total')).toBeInTheDocument()
  })

  it('renders duration', () => {
    render(<TestResults results={mockResults} />)
    expect(screen.getByText('50ms')).toBeInTheDocument()
  })

  it('renders all test names', () => {
    render(<TestResults results={mockResults} />)

    expect(screen.getByText('passing-test')).toBeInTheDocument()
    expect(screen.getByText('another-passing')).toBeInTheDocument()
    expect(screen.getByText('failing-test')).toBeInTheDocument()
    expect(screen.getByText('skipped-test')).toBeInTheDocument()
  })

  it('renders passing tests with checkmark', () => {
    render(<TestResults results={mockResults} />)

    // Count checkmarks (✓)
    const checkmarks = screen.getAllByText('✓')
    expect(checkmarks).toHaveLength(2)
  })

  it('renders failing tests with X', () => {
    render(<TestResults results={mockResults} />)

    const xmarks = screen.getAllByText('✗')
    expect(xmarks).toHaveLength(1)
  })

  it('renders skipped tests with circle', () => {
    render(<TestResults results={mockResults} />)

    const circles = screen.getAllByText('○')
    expect(circles).toHaveLength(1)
  })

  it('renders failure message', () => {
    render(<TestResults results={mockResults} />)
    expect(screen.getByText('expected output to contain "hello"')).toBeInTheDocument()
  })

  it('renders re-run button when results exist', () => {
    const onRunTests = vi.fn()
    render(<TestResults results={mockResults} onRunTests={onRunTests} />)

    const button = screen.getByRole('button', { name: /re-run/i })
    expect(button).toBeInTheDocument()

    fireEvent.click(button)
    expect(onRunTests).toHaveBeenCalled()
  })

  it('renders test durations', () => {
    render(<TestResults results={mockResults} />)

    expect(screen.getByText('10ms')).toBeInTheDocument()
    expect(screen.getByText('12ms')).toBeInTheDocument()
    expect(screen.getByText('15ms')).toBeInTheDocument()
  })

  it('renders error message when test has error', () => {
    const resultsWithError: SuiteResult = {
      ...mockResults,
      results: [
        {
          testName: 'error-test',
          passed: false,
          skipped: false,
          durationMs: 5,
          error: 'Template parsing failed',
        },
      ],
    }

    render(<TestResults results={resultsWithError} />)
    expect(screen.getByText('Template parsing failed')).toBeInTheDocument()
  })

  it('does not show skipped count when zero', () => {
    const noSkipped: SuiteResult = {
      ...mockResults,
      skipped: 0,
      results: mockResults.results.filter(r => !r.skipped),
    }

    render(<TestResults results={noSkipped} />)
    expect(screen.queryByText(/skipped/i)).not.toBeInTheDocument()
  })

  it('does not show failed count when zero', () => {
    const noneFailedResults: SuiteResult = {
      ...mockResults,
      failed: 0,
      results: mockResults.results.filter(r => r.passed || r.skipped),
    }

    render(<TestResults results={noneFailedResults} />)
    // Should show "2 passed" but not "0 failed"
    expect(screen.getByText('2 passed')).toBeInTheDocument()
    expect(screen.queryByText(/\d+ failed/)).not.toBeInTheDocument()
  })
})
