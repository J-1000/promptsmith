import { render, screen } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
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
})
