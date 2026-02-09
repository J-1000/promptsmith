import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PlaygroundPage } from './PlaygroundPage'

vi.mock('../api', () => ({
  listPrompts: vi.fn(),
  getPromptVersions: vi.fn(),
  getAvailableModels: vi.fn(),
  runPlayground: vi.fn(),
}))

import { listPrompts, getPromptVersions, getAvailableModels, runPlayground } from '../api'

const mockPrompts = [
  { id: '1', name: 'greeting', description: 'A greeting', file_path: 'prompts/greeting.prompt', version: '1.0.0', created_at: '2024-01-15T00:00:00Z' },
  { id: '2', name: 'summary', description: 'Summarizer', file_path: 'prompts/summary.prompt', version: '2.0.0', created_at: '2024-01-16T00:00:00Z' },
]

const mockVersions = [
  { id: 'v1', version: '1.0.0', content: 'Hello {{name}}!', commit_message: 'Initial', created_at: '2024-01-15T00:00:00Z', tags: [] },
  { id: 'v2', version: '1.0.1', content: 'Hi {{name}}, welcome!', commit_message: 'Update', created_at: '2024-01-16T00:00:00Z', tags: [] },
]

const mockModels = [
  { id: 'gpt-4o-mini', provider: 'openai' },
  { id: 'gpt-4o', provider: 'openai' },
  { id: 'claude-sonnet', provider: 'anthropic' },
]

const mockRunResult = {
  output: 'Hello World! Nice to meet you.',
  rendered_prompt: 'Hello World!',
  model: 'gpt-4o-mini',
  prompt_tokens: 10,
  output_tokens: 8,
  latency_ms: 342,
  cost: 0.0001,
}

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/playground']}>
      <Routes>
        <Route path="/playground" element={<PlaygroundPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('PlaygroundPage', () => {
  beforeEach(() => {
    vi.mocked(listPrompts).mockResolvedValue(mockPrompts)
    vi.mocked(getAvailableModels).mockResolvedValue(mockModels)
    vi.mocked(getPromptVersions).mockResolvedValue(mockVersions)
    vi.mocked(runPlayground).mockResolvedValue(mockRunResult)
  })

  it('renders the playground title', async () => {
    renderPage()
    expect(screen.getByText('Playground')).toBeInTheDocument()
  })

  it('shows input and output panels', () => {
    renderPage()
    expect(screen.getByText('Input')).toBeInTheDocument()
    expect(screen.getByText('Output')).toBeInTheDocument()
  })

  it('shows source toggle buttons', () => {
    renderPage()
    expect(screen.getByText('From Library')).toBeInTheDocument()
    expect(screen.getByText('Ad-hoc')).toBeInTheDocument()
  })

  it('defaults to ad-hoc mode with textarea', () => {
    renderPage()
    expect(screen.getByPlaceholderText(/enter your prompt/i)).toBeInTheDocument()
  })

  it('switches to library mode and shows prompt dropdown', async () => {
    renderPage()
    fireEvent.click(screen.getByText('From Library'))
    await waitFor(() => {
      expect(screen.getByText('Select a prompt...')).toBeInTheDocument()
    })
  })

  it('loads models into the model selector', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByDisplayValue('gpt-4o-mini')).toBeInTheDocument()
    })
  })

  it('shows temperature slider', () => {
    renderPage()
    expect(screen.getByText('Temperature')).toBeInTheDocument()
    expect(screen.getByText('1.0')).toBeInTheDocument()
  })

  it('shows max tokens input', () => {
    renderPage()
    expect(screen.getByText('Max Tokens')).toBeInTheDocument()
    expect(screen.getByDisplayValue('1024')).toBeInTheDocument()
  })

  it('shows empty state when no run has been executed', () => {
    renderPage()
    expect(screen.getByText('Run a prompt to see output')).toBeInTheDocument()
  })

  it('disables run button when ad-hoc content is empty', () => {
    renderPage()
    const runBtn = screen.getByRole('button', { name: 'Run' })
    expect(runBtn).toBeDisabled()
  })

  it('enables run button when ad-hoc content is provided', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByDisplayValue('gpt-4o-mini')).toBeInTheDocument()
    })
    fireEvent.change(screen.getByPlaceholderText(/enter your prompt/i), {
      target: { value: 'Test prompt' },
    })
    const runBtn = screen.getByRole('button', { name: 'Run' })
    expect(runBtn).not.toBeDisabled()
  })

  it('detects variables from ad-hoc content', async () => {
    renderPage()
    fireEvent.change(screen.getByPlaceholderText(/enter your prompt/i), {
      target: { value: 'Hello {{name}}, welcome to {{place}}!' },
    })
    await waitFor(() => {
      expect(screen.getByText('{{name}}')).toBeInTheDocument()
      expect(screen.getByText('{{place}}')).toBeInTheDocument()
    })
  })

  it('runs playground and displays output', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByDisplayValue('gpt-4o-mini')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByPlaceholderText(/enter your prompt/i), {
      target: { value: 'Hello World!' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await waitFor(() => {
      expect(screen.getByText('Hello World! Nice to meet you.')).toBeInTheDocument()
    })
  })

  it('shows stats after a successful run', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByDisplayValue('gpt-4o-mini')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByPlaceholderText(/enter your prompt/i), {
      target: { value: 'Test' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await waitFor(() => {
      expect(screen.getByText('342ms')).toBeInTheDocument()
      expect(screen.getByText('10')).toBeInTheDocument()
      expect(screen.getByText('8')).toBeInTheDocument()
      expect(screen.getByText('$0.0001')).toBeInTheDocument()
    })
  })

  it('shows error when run fails', async () => {
    vi.mocked(runPlayground).mockRejectedValue(new Error('API key missing'))
    renderPage()
    await waitFor(() => {
      expect(screen.getByDisplayValue('gpt-4o-mini')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByPlaceholderText(/enter your prompt/i), {
      target: { value: 'Test' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await waitFor(() => {
      expect(screen.getByText('API key missing')).toBeInTheDocument()
    })
  })

  it('loads versions when a library prompt is selected', async () => {
    renderPage()
    fireEvent.click(screen.getByText('From Library'))

    await waitFor(() => {
      expect(screen.getByText('Select a prompt...')).toBeInTheDocument()
    })

    const select = screen.getByDisplayValue('Select a prompt...')
    fireEvent.change(select, { target: { value: 'greeting' } })

    await waitFor(() => {
      expect(getPromptVersions).toHaveBeenCalledWith('greeting')
    })
  })

  it('shows rendered prompt toggle after run', async () => {
    renderPage()
    await waitFor(() => {
      expect(screen.getByDisplayValue('gpt-4o-mini')).toBeInTheDocument()
    })

    fireEvent.change(screen.getByPlaceholderText(/enter your prompt/i), {
      target: { value: 'Test' },
    })
    fireEvent.click(screen.getByRole('button', { name: 'Run' }))

    await waitFor(() => {
      expect(screen.getByText('Show rendered prompt')).toBeInTheDocument()
    })
  })
})
