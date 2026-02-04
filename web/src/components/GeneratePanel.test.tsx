import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { GeneratePanel, GenerateResult } from './GeneratePanel'

const mockResults: GenerateResult = {
  original: 'Original prompt text',
  variations: [
    {
      content: 'First variation content',
      description: 'More concise version',
      tokenDelta: -10,
    },
    {
      content: 'Second variation content',
      description: 'Added context',
      tokenDelta: 5,
    },
    {
      content: 'Third variation content',
      description: 'Restructured',
    },
  ],
  model: 'gpt-4o-mini',
  type: 'variations',
  goal: 'be concise',
}

describe('GeneratePanel', () => {
  it('renders empty state when no results', () => {
    render(<GeneratePanel results={null} />)
    expect(screen.getByText('Generate variations of your prompt using AI')).toBeInTheDocument()
  })

  it('renders type selection buttons', () => {
    render(<GeneratePanel results={null} />)
    expect(screen.getByRole('button', { name: 'variations' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'compress' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'expand' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'rephrase' })).toBeInTheDocument()
  })

  it('renders count selector', () => {
    render(<GeneratePanel results={null} />)
    const select = screen.getByRole('combobox')
    expect(select).toBeInTheDocument()
    expect(select).toHaveValue('3') // default
  })

  it('renders goal input', () => {
    render(<GeneratePanel results={null} />)
    expect(screen.getByPlaceholderText(/be more concise/i)).toBeInTheDocument()
  })

  it('renders generate button', () => {
    render(<GeneratePanel results={null} />)
    expect(screen.getByRole('button', { name: 'Generate' })).toBeInTheDocument()
  })

  it('calls onGenerate with correct params', () => {
    const onGenerate = vi.fn()
    render(<GeneratePanel results={null} onGenerate={onGenerate} />)

    // Click compress type
    fireEvent.click(screen.getByRole('button', { name: 'compress' }))

    // Change count
    fireEvent.change(screen.getByRole('combobox'), { target: { value: '5' } })

    // Set goal
    fireEvent.change(screen.getByPlaceholderText(/be more concise/i), {
      target: { value: 'reduce tokens' },
    })

    // Generate
    fireEvent.click(screen.getByRole('button', { name: 'Generate' }))

    expect(onGenerate).toHaveBeenCalledWith('compress', 5, 'reduce tokens')
  })

  it('shows generating state', () => {
    render(<GeneratePanel results={null} isGenerating />)
    const button = screen.getByRole('button', { name: 'Generating...' })
    expect(button).toBeDisabled()
  })

  it('renders results when provided', () => {
    render(<GeneratePanel results={mockResults} />)
    expect(screen.getByText('3 variations generated')).toBeInTheDocument()
    expect(screen.getByText('using gpt-4o-mini')).toBeInTheDocument()
  })

  it('renders variation descriptions', () => {
    render(<GeneratePanel results={mockResults} />)
    expect(screen.getByText('More concise version')).toBeInTheDocument()
    expect(screen.getByText('Added context')).toBeInTheDocument()
    expect(screen.getByText('Restructured')).toBeInTheDocument()
  })

  it('shows token delta for variations', () => {
    render(<GeneratePanel results={mockResults} />)
    expect(screen.getByText('-10 tokens')).toBeInTheDocument()
    expect(screen.getByText('+5 tokens')).toBeInTheDocument()
  })

  it('expands variation on click', () => {
    render(<GeneratePanel results={mockResults} />)

    // Click first variation header
    fireEvent.click(screen.getByText('More concise version'))

    // Content should be visible
    expect(screen.getByText('First variation content')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Copy' })).toBeInTheDocument()
  })

  it('collapses variation when clicked again', () => {
    render(<GeneratePanel results={mockResults} />)

    // Click to expand
    fireEvent.click(screen.getByText('More concise version'))
    expect(screen.getByText('First variation content')).toBeInTheDocument()

    // Click to collapse
    fireEvent.click(screen.getByText('More concise version'))
    expect(screen.queryByText('First variation content')).not.toBeInTheDocument()
  })

  it('renders variation numbers', () => {
    render(<GeneratePanel results={mockResults} />)
    expect(screen.getByText('#1')).toBeInTheDocument()
    expect(screen.getByText('#2')).toBeInTheDocument()
    expect(screen.getByText('#3')).toBeInTheDocument()
  })

  it('handles empty variations array', () => {
    const emptyResults: GenerateResult = {
      ...mockResults,
      variations: [],
    }
    render(<GeneratePanel results={emptyResults} />)
    // With empty variations, shows the empty state (no results section)
    // Should still show controls but not results header
    expect(screen.queryByText('0 variations generated')).not.toBeInTheDocument()
  })
})
