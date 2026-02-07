import { useState } from 'react'
import styles from './DiffViewer.module.css'

export interface DiffComment {
  id: string
  lineNumber: number
  content: string
  createdAt: string
}

interface DiffViewerProps {
  oldVersion: string
  newVersion: string
  diff: string
  comments?: DiffComment[]
  onAddComment?: (lineNumber: number, content: string) => void
  onDeleteComment?: (commentId: string) => void
}

export function DiffViewer({ oldVersion, newVersion, diff, comments = [], onAddComment, onDeleteComment }: DiffViewerProps) {
  const lines = diff.split('\n')
  const [commentLine, setCommentLine] = useState<number | null>(null)
  const [commentText, setCommentText] = useState('')

  const commentsByLine: Record<number, DiffComment[]> = {}
  for (const c of comments) {
    if (!commentsByLine[c.lineNumber]) commentsByLine[c.lineNumber] = []
    commentsByLine[c.lineNumber].push(c)
  }

  const handleSubmitComment = () => {
    if (commentLine === null || !commentText.trim() || !onAddComment) return
    onAddComment(commentLine, commentText.trim())
    setCommentLine(null)
    setCommentText('')
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <span className={styles.headerText}>
          Comparing <code>v{oldVersion}</code> with <code>v{newVersion}</code>
        </span>
      </div>
      <div className={styles.diffContent}>
        {lines.map((line, index) => {
          let lineClass = styles.line
          let prefix = ' '
          const lineNum = index + 1

          if (line.startsWith('@@')) {
            lineClass = `${styles.line} ${styles.hunk}`
            return (
              <div key={index} className={lineClass}>
                <span className={styles.lineNumber}></span>
                <span className={styles.lineContent}>{line}</span>
              </div>
            )
          } else if (line.startsWith('+')) {
            lineClass = `${styles.line} ${styles.added}`
            prefix = '+'
          } else if (line.startsWith('-')) {
            lineClass = `${styles.line} ${styles.removed}`
            prefix = '-'
          }

          return (
            <div key={index}>
              <div className={lineClass}>
                <span
                  className={`${styles.lineNumber} ${onAddComment ? styles.lineNumberClickable : ''}`}
                  onClick={() => onAddComment && setCommentLine(commentLine === lineNum ? null : lineNum)}
                  title={onAddComment ? 'Add comment' : undefined}
                >
                  {lineNum}
                </span>
                <span className={styles.linePrefix}>{prefix}</span>
                <span className={styles.lineContent}>
                  {line.startsWith('+') || line.startsWith('-') ? line.slice(1) : line}
                </span>
              </div>
              {commentsByLine[lineNum]?.map((c) => (
                <div key={c.id} className={styles.commentBubble}>
                  <span className={styles.commentContent}>{c.content}</span>
                  {onDeleteComment && (
                    <button className={styles.commentDelete} onClick={() => onDeleteComment(c.id)} aria-label="Delete comment">&times;</button>
                  )}
                </div>
              ))}
              {commentLine === lineNum && (
                <div className={styles.commentInput}>
                  <textarea
                    className={styles.commentTextarea}
                    value={commentText}
                    onChange={(e) => setCommentText(e.target.value)}
                    placeholder="Add a comment..."
                    rows={2}
                    autoFocus
                  />
                  <div className={styles.commentActions}>
                    <button className={styles.commentSubmit} onClick={handleSubmitComment} disabled={!commentText.trim()}>Comment</button>
                    <button className={styles.commentCancel} onClick={() => { setCommentLine(null); setCommentText('') }}>Cancel</button>
                  </div>
                </div>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}
