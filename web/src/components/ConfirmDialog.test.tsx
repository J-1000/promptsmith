import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { ConfirmDialog } from './ConfirmDialog'

describe('ConfirmDialog', () => {
  it('renders title and message', () => {
    render(
      <ConfirmDialog
        title="Delete Prompt"
        message="Are you sure?"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    )
    expect(screen.getByText('Delete Prompt')).toBeInTheDocument()
    expect(screen.getByText('Are you sure?')).toBeInTheDocument()
  })

  it('renders with role dialog', () => {
    render(
      <ConfirmDialog
        title="Confirm"
        message="Sure?"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    )
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })

  it('calls onConfirm when confirm button clicked', async () => {
    const user = userEvent.setup()
    const onConfirm = vi.fn()
    render(
      <ConfirmDialog
        title="Delete"
        message="Sure?"
        onConfirm={onConfirm}
        onCancel={() => {}}
      />
    )
    await user.click(screen.getByText('Confirm'))
    expect(onConfirm).toHaveBeenCalled()
  })

  it('calls onCancel when cancel button clicked', async () => {
    const user = userEvent.setup()
    const onCancel = vi.fn()
    render(
      <ConfirmDialog
        title="Delete"
        message="Sure?"
        onConfirm={() => {}}
        onCancel={onCancel}
      />
    )
    await user.click(screen.getByText('Cancel'))
    expect(onCancel).toHaveBeenCalled()
  })

  it('calls onCancel when overlay clicked', async () => {
    const user = userEvent.setup()
    const onCancel = vi.fn()
    render(
      <ConfirmDialog
        title="Delete"
        message="Sure?"
        onConfirm={() => {}}
        onCancel={onCancel}
      />
    )
    await user.click(screen.getByTestId('confirm-overlay'))
    expect(onCancel).toHaveBeenCalled()
  })

  it('uses custom button labels', () => {
    render(
      <ConfirmDialog
        title="Delete"
        message="Sure?"
        confirmLabel="Yes, delete"
        cancelLabel="No, keep"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    )
    expect(screen.getByText('Yes, delete')).toBeInTheDocument()
    expect(screen.getByText('No, keep')).toBeInTheDocument()
  })

  it('renders danger variant', () => {
    render(
      <ConfirmDialog
        title="Delete"
        message="Sure?"
        variant="danger"
        onConfirm={() => {}}
        onCancel={() => {}}
      />
    )
    expect(screen.getByText('Confirm')).toBeInTheDocument()
  })
})
