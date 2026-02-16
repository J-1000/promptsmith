import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ChainDetailPage } from './ChainDetailPage'

vi.mock('../api', () => ({
  getChain: vi.fn(),
  deleteChain: vi.fn(),
  saveChainSteps: vi.fn(),
  runChain: vi.fn(),
  listPrompts: vi.fn(),
  getAvailableModels: vi.fn(),
}))

import { getChain, deleteChain, saveChainSteps, runChain, listPrompts, getAvailableModels } from '../api'

const mockChain = {
  id: '1',
  name: 'summarize-translate',
  description: 'Summarize then translate',
  steps: [
    {
      id: 's1',
      step_order: 1,
      prompt_name: 'summarize',
      input_mapping: { text: '{{input.text}}' },
      output_key: 'summary',
    },
    {
      id: 's2',
      step_order: 2,
      prompt_name: 'translate',
      input_mapping: { text: '{{steps.summary.output}}' },
      output_key: 'translation',
    },
  ],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-02T00:00:00Z',
}

const mockPrompts = [
  { id: 'p1', name: 'summarize', description: 'Summarizer', file_path: 'prompts/summarize.prompt', created_at: '' },
  { id: 'p2', name: 'translate', description: 'Translator', file_path: 'prompts/translate.prompt', created_at: '' },
]

const mockModels = [
  { id: 'gpt-4o-mini', provider: 'openai' },
  { id: 'gpt-4o', provider: 'openai' },
]

function renderWithRouter(name: string) {
  return render(
    <MemoryRouter initialEntries={[`/chains/${name}`]}>
      <Routes>
        <Route path="/chains/:name" element={<ChainDetailPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('ChainDetailPage', () => {
  beforeEach(() => {
    vi.mocked(getChain).mockResolvedValue(mockChain)
    vi.mocked(listPrompts).mockResolvedValue(mockPrompts)
    vi.mocked(getAvailableModels).mockResolvedValue(mockModels)
    vi.mocked(saveChainSteps).mockResolvedValue([])
    vi.mocked(runChain).mockResolvedValue({
      id: 'r1',
      status: 'completed',
      inputs: { text: 'hello' },
      results: [
        {
          step_order: 1,
          prompt_name: 'summarize',
          output_key: 'summary',
          rendered_prompt: 'Summarize: hello',
          output: 'Summary of hello',
          duration_ms: 500,
        },
      ],
      final_output: 'Summary of hello',
      started_at: '2024-01-01T00:00:00Z',
      completed_at: '2024-01-01T00:00:01Z',
    })
    vi.mocked(deleteChain).mockResolvedValue()
  })

  it('shows loading state', () => {
    renderWithRouter('summarize-translate')
    expect(screen.getByText(/loading chain/i)).toBeInTheDocument()
  })

  it('renders chain name', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'summarize-translate' })).toBeInTheDocument()
    })
  })

  it('renders chain description', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText('Summarize then translate')).toBeInTheDocument()
    })
  })

  it('renders breadcrumb', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText('Chains')).toBeInTheDocument()
    })
  })

  it('shows steps section', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText('Steps')).toBeInTheDocument()
    })
  })

  it('renders step cards with step numbers', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText('Step 1')).toBeInTheDocument()
      expect(screen.getByText('Step 2')).toBeInTheDocument()
    })
  })

  it('shows delete button', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText('Delete')).toBeInTheDocument()
    })
  })

  it('shows add step button', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText('+ Add Step')).toBeInTheDocument()
    })
  })

  it('shows run chain section', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Run Chain' })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /run chain/i })).toBeInTheDocument()
    })
  })

  it('auto-detects input fields from mappings', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText('text')).toBeInTheDocument()
      expect(screen.getByPlaceholderText('Enter text...')).toBeInTheDocument()
    })
  })

  it('shows error state on API failure', async () => {
    vi.mocked(getChain).mockRejectedValue(new Error('Not found'))
    renderWithRouter('nonexistent')
    await waitFor(() => {
      expect(screen.getByText(/failed to load chain/i)).toBeInTheDocument()
    })
  })

  it('shows empty steps message for chain with no steps', async () => {
    vi.mocked(getChain).mockResolvedValue({
      ...mockChain,
      steps: [],
    })
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      expect(screen.getByText(/no steps configured/i)).toBeInTheDocument()
    })
  })

  it('shows model selector', async () => {
    renderWithRouter('summarize-translate')
    await waitFor(() => {
      const select = screen.getByDisplayValue('gpt-4o-mini')
      expect(select).toBeInTheDocument()
    })
  })

  it('runs chain and shows results', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter('summarize-translate')

    await waitFor(() => {
      expect(screen.getByText('Run Chain', { selector: 'button' })).toBeInTheDocument()
    })

    // Fill input
    const input = screen.getByPlaceholderText('Enter text...')
    await user.type(input, 'hello world')

    // Click run
    await user.click(screen.getByText('Run Chain', { selector: 'button' }))

    await waitFor(() => {
      expect(screen.getByText('Final Output')).toBeInTheDocument()
      expect(screen.getByText('Summary of hello')).toBeInTheDocument()
    })
  })

  it('shows save button after editing steps', async () => {
    const user = (await import('@testing-library/user-event')).default.setup()
    renderWithRouter('summarize-translate')

    await waitFor(() => {
      expect(screen.getByText('+ Add Step')).toBeInTheDocument()
    })

    await user.click(screen.getByText('+ Add Step'))

    expect(screen.getByText('Save Steps')).toBeInTheDocument()
  })
})
