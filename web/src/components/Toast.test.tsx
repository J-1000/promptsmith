import { render, screen, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { Toast } from './Toast'

describe('Toast', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders message', () => {
    render(<Toast message="Prompt created" onClose={() => {}} />)
    expect(screen.getByText('Prompt created')).toBeInTheDocument()
  })

  it('renders with role alert', () => {
    render(<Toast message="Done" onClose={() => {}} />)
    expect(screen.getByRole('alert')).toBeInTheDocument()
  })

  it('calls onClose when close button clicked', async () => {
    vi.useRealTimers()
    const user = userEvent.setup()
    const onClose = vi.fn()
    render(<Toast message="Done" onClose={onClose} />)
    await user.click(screen.getByLabelText('Close'))
    expect(onClose).toHaveBeenCalled()
  })

  it('auto-dismisses after 3 seconds', () => {
    const onClose = vi.fn()
    render(<Toast message="Done" onClose={onClose} />)
    expect(onClose).not.toHaveBeenCalled()
    act(() => {
      vi.advanceTimersByTime(3000)
    })
    expect(onClose).toHaveBeenCalled()
  })

  it('renders different types', () => {
    const { rerender } = render(<Toast message="OK" type="success" onClose={() => {}} />)
    expect(screen.getByRole('alert')).toBeInTheDocument()

    rerender(<Toast message="Fail" type="error" onClose={() => {}} />)
    expect(screen.getByText('Fail')).toBeInTheDocument()

    rerender(<Toast message="FYI" type="info" onClose={() => {}} />)
    expect(screen.getByText('FYI')).toBeInTheDocument()
  })
})
