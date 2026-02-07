import { render, screen, fireEvent } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { DiffViewer } from './DiffViewer'

describe('DiffViewer', () => {
  const mockDiff = `@@ -1,3 +1,4 @@
 context line
-removed line
+added line
+another added line`

  it('renders version comparison header', () => {
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} />)
    expect(screen.getByText(/v1\.0\.0/)).toBeInTheDocument()
    expect(screen.getByText(/v1\.0\.1/)).toBeInTheDocument()
  })

  it('renders hunk header', () => {
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} />)
    expect(screen.getByText(/@@ -1,3 \+1,4 @@/)).toBeInTheDocument()
  })

  it('renders added lines', () => {
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} />)
    expect(screen.getByText('added line')).toBeInTheDocument()
    expect(screen.getByText('another added line')).toBeInTheDocument()
  })

  it('renders removed lines', () => {
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} />)
    expect(screen.getByText('removed line')).toBeInTheDocument()
  })

  it('renders context lines', () => {
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} />)
    expect(screen.getByText('context line')).toBeInTheDocument()
  })

  it('renders inline comments', () => {
    const comments = [
      { id: 'c1', lineNumber: 2, content: 'This line looks wrong', createdAt: '2024-01-01T00:00:00Z' },
    ]
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} comments={comments} />)
    expect(screen.getByText('This line looks wrong')).toBeInTheDocument()
  })

  it('shows comment input when line number clicked', () => {
    const onAddComment = vi.fn()
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} onAddComment={onAddComment} />)
    fireEvent.click(screen.getByText('2'))
    expect(screen.getByPlaceholderText('Add a comment...')).toBeInTheDocument()
  })

  it('calls onAddComment when submitting', () => {
    const onAddComment = vi.fn()
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} onAddComment={onAddComment} />)
    fireEvent.click(screen.getByText('2'))
    fireEvent.change(screen.getByPlaceholderText('Add a comment...'), { target: { value: 'My comment' } })
    fireEvent.click(screen.getByText('Comment'))
    expect(onAddComment).toHaveBeenCalledWith(2, 'My comment')
  })

  it('shows delete button on comments when callback provided', () => {
    const onDeleteComment = vi.fn()
    const comments = [
      { id: 'c1', lineNumber: 2, content: 'Test comment', createdAt: '2024-01-01T00:00:00Z' },
    ]
    render(<DiffViewer oldVersion="1.0.0" newVersion="1.0.1" diff={mockDiff} comments={comments} onDeleteComment={onDeleteComment} />)
    fireEvent.click(screen.getByLabelText('Delete comment'))
    expect(onDeleteComment).toHaveBeenCalledWith('c1')
  })
})
