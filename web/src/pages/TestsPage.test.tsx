import { render, screen, waitFor } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { TestsPage } from './TestsPage'

vi.mock('../api', () => ({
  listTests: vi.fn(),
  listTestRuns: vi.fn(),
}))

import { listTests, listTestRuns } from '../api'

const mockSuites = [
  {
    name: 'greeting-tests',
    file_path: 'tests/greeting.test.yaml',
    prompt: 'greeting',
    description: 'Tests for greeting prompt',
    test_count: 5,
  },
  {
    name: 'summarize-tests',
    file_path: 'tests/summarize.test.yaml',
    prompt: 'summarize',
    test_count: 3,
  },
]

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('TestsPage', () => {
  beforeEach(() => {
    vi.mocked(listTests).mockResolvedValue(mockSuites)
    vi.mocked(listTestRuns).mockResolvedValue([])
  })

  it('renders the page title', async () => {
    renderWithRouter(<TestsPage />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /test suites/i })).toBeInTheDocument()
    })
  })

  it('shows suite count', async () => {
    renderWithRouter(<TestsPage />)
    await waitFor(() => {
      expect(screen.getByText(/2 test suites configured/i)).toBeInTheDocument()
    })
  })

  it('renders suite cards', async () => {
    renderWithRouter(<TestsPage />)
    await waitFor(() => {
      expect(screen.getByText('greeting-tests')).toBeInTheDocument()
      expect(screen.getByText('summarize-tests')).toBeInTheDocument()
    })
  })

  it('shows test count badges', async () => {
    renderWithRouter(<TestsPage />)
    await waitFor(() => {
      expect(screen.getByText('5 tests')).toBeInTheDocument()
      expect(screen.getByText('3 tests')).toBeInTheDocument()
    })
  })

  it('shows loading state', () => {
    renderWithRouter(<TestsPage />)
    expect(screen.getByText(/loading test suites/i)).toBeInTheDocument()
  })

  it('shows error state on API failure', async () => {
    vi.mocked(listTests).mockRejectedValue(new Error('Network error'))
    renderWithRouter(<TestsPage />)
    await waitFor(() => {
      expect(screen.getByText(/failed to load tests/i)).toBeInTheDocument()
    })
  })

  it('shows empty state when no suites', async () => {
    vi.mocked(listTests).mockResolvedValue([])
    renderWithRouter(<TestsPage />)
    await waitFor(() => {
      expect(screen.getByText(/no test suites found/i)).toBeInTheDocument()
    })
  })

  it('shows flaky badge for suites with high flakiness', async () => {
    const runs = [
      { id: '1', suite_id: 'greeting-tests', status: 'passed', results: {} as never, started_at: '', completed_at: '' },
      { id: '2', suite_id: 'greeting-tests', status: 'failed', results: {} as never, started_at: '', completed_at: '' },
      { id: '3', suite_id: 'greeting-tests', status: 'passed', results: {} as never, started_at: '', completed_at: '' },
      { id: '4', suite_id: 'greeting-tests', status: 'failed', results: {} as never, started_at: '', completed_at: '' },
    ]
    vi.mocked(listTestRuns).mockImplementation((name: string) =>
      name === 'greeting-tests' ? Promise.resolve(runs) : Promise.resolve([])
    )
    renderWithRouter(<TestsPage />)
    await waitFor(() => {
      expect(screen.getByText(/flaky/i)).toBeInTheDocument()
    }, { timeout: 3000 })
  })

  it('filters suites by search', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter(<TestsPage />)

    await waitFor(() => {
      expect(screen.getByText('greeting-tests')).toBeInTheDocument()
    })

    const searchInput = screen.getByPlaceholderText(/search test suites/i)
    await user.type(searchInput, 'greeting')

    expect(screen.getByText('greeting-tests')).toBeInTheDocument()
    expect(screen.queryByText('summarize-tests')).not.toBeInTheDocument()
  })
})
