import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { EditorPage } from './EditorPage'

vi.mock('../api', () => ({
  getPrompt: vi.fn(),
  getPromptVersions: vi.fn(),
  createVersion: vi.fn(),
}))

import { getPrompt, getPromptVersions } from '../api'

const mockPrompt = {
  id: '1',
  name: 'greeting',
  description: 'A greeting prompt',
  file_path: 'prompts/greeting.prompt',
  version: '1.0.0',
  created_at: '2024-01-15T00:00:00Z',
}

const mockVersions = [
  {
    id: 'v1',
    version: '1.0.0',
    content: 'Hello {{name}}, welcome to {{place}}!',
    commit_message: 'Initial version',
    created_at: '2024-01-15T00:00:00Z',
    tags: [],
  },
]

function renderWithRoute() {
  return render(
    <MemoryRouter initialEntries={['/prompt/greeting/edit']}>
      <Routes>
        <Route path="/prompt/:name/edit" element={<EditorPage />} />
      </Routes>
    </MemoryRouter>
  )
}

describe('EditorPage', () => {
  beforeEach(() => {
    vi.mocked(getPrompt).mockResolvedValue(mockPrompt)
    vi.mocked(getPromptVersions).mockResolvedValue(mockVersions)
  })

  it('shows loading state initially', () => {
    renderWithRoute()
    expect(screen.getByText(/loading editor/i)).toBeInTheDocument()
  })

  it('renders the editor with prompt content', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText(/edit greeting/i)).toBeInTheDocument()
    })
    const textarea = screen.getByPlaceholderText(/enter your prompt content/i) as HTMLTextAreaElement
    expect(textarea.value).toContain('Hello {{name}}')
  })

  it('extracts and displays variables', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('name')).toBeInTheDocument()
      expect(screen.getByText('place')).toBeInTheDocument()
    })
  })

  it('shows token count estimate', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText(/tokens \(est\.\)/i)).toBeInTheDocument()
    })
  })

  it('shows version number', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('v1.0.0')).toBeInTheDocument()
    })
  })

  it('has breadcrumb navigation', async () => {
    renderWithRoute()
    await waitFor(() => {
      expect(screen.getByText('Edit')).toBeInTheDocument()
    })
  })

  it('shows save button disabled when no changes', async () => {
    renderWithRoute()
    await waitFor(() => {
      const saveBtn = screen.getByRole('button', { name: /save version/i })
      expect(saveBtn).toBeDisabled()
    })
  })
})
