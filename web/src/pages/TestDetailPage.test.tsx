import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { TestDetailPage } from './TestDetailPage'

vi.mock('../api', () => ({
  getTest: vi.fn(),
  runTest: vi.fn(),
}))

import { getTest } from '../api'

const mockSuite = {
  name: 'greeting-tests',
  file_path: 'tests/greeting.test.yaml',
  prompt: 'greeting',
  description: 'Tests for the greeting prompt',
  test_count: 3,
}

function renderWithRoute() {
  return render(
    <MemoryRouter initialEntries={['/tests/greeting-tests']}>
      <Routes>
        <Route path="/tests/:name" element={<TestDetailPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('TestDetailPage', () => {
  beforeEach(() => {
    vi.mocked(getTest).mockResolvedValue(mockSuite)
  })

  it('shows loading state initially', () => {
    renderWithRoute()
    expect(screen.getByText(/loading test suite/i)).toBeInTheDocument()
  })

  it('renders suite name', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: /greeting-tests/i })).toBeInTheDocument()
    })
  })

  it('shows test count badge', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('3 tests')).toBeInTheDocument()
    })
  })

  it('shows link to associated prompt', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('greeting')).toBeInTheDocument()
    })
  })

  it('shows description', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('Tests for the greeting prompt')).toBeInTheDocument()
    })
  })

  it('has breadcrumb navigation', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('Tests')).toBeInTheDocument()
    })
  })

  it('shows run tests button', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByRole('button', { name: /run tests/i })).toBeInTheDocument()
    })
  })

  it('shows error state on API failure', async () => {
    vi.mocked(getTest).mockRejectedValue(new Error('Not found'))
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText(/failed to load test suite/i)).toBeInTheDocument()
    })
  })
})
