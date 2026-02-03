import styles from './DiffViewer.module.css'

interface DiffViewerProps {
  oldVersion: string
  newVersion: string
  diff: string
}

export function DiffViewer({ oldVersion, newVersion, diff }: DiffViewerProps) {
  const lines = diff.split('\n')

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
            <div key={index} className={lineClass}>
              <span className={styles.lineNumber}>{index + 1}</span>
              <span className={styles.linePrefix}>{prefix}</span>
              <span className={styles.lineContent}>
                {line.startsWith('+') || line.startsWith('-') ? line.slice(1) : line}
              </span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
