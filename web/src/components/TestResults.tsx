import styles from './TestResults.module.css'

export interface TestResult {
  testName: string
  passed: boolean
  skipped: boolean
  output?: string
  failures?: Array<{
    type: string
    message?: string
    expected: string
    actual: string
  }>
  error?: string
  durationMs: number
}

export interface SuiteResult {
  suiteName: string
  promptName: string
  version: string
  passed: number
  failed: number
  skipped: number
  total: number
  results: TestResult[]
  durationMs: number
}

interface TestResultsProps {
  results: SuiteResult | null
  onRunTests?: () => void
  isRunning?: boolean
}

export function TestResults({ results, onRunTests, isRunning }: TestResultsProps) {
  if (!results) {
    return (
      <div className={styles.empty}>
        <div className={styles.emptyIcon}>&#128203;</div>
        <p className={styles.emptyText}>No test results yet</p>
        {onRunTests && (
          <button
            className={styles.runButton}
            onClick={onRunTests}
            disabled={isRunning}
          >
            {isRunning ? 'Running...' : 'Run Tests'}
          </button>
        )}
      </div>
    )
  }

  const { passed, failed, skipped, total, durationMs } = results

  return (
    <div className={styles.container}>
      <div className={styles.summary}>
        <div className={styles.summaryStats}>
          <span className={styles.statPassed}>{passed} passed</span>
          {failed > 0 && <span className={styles.statFailed}>{failed} failed</span>}
          {skipped > 0 && <span className={styles.statSkipped}>{skipped} skipped</span>}
          <span className={styles.statTotal}>{total} total</span>
        </div>
        <div className={styles.summaryMeta}>
          <span className={styles.duration}>{durationMs}ms</span>
          {onRunTests && (
            <button
              className={styles.rerunButton}
              onClick={onRunTests}
              disabled={isRunning}
            >
              {isRunning ? 'Running...' : 'Re-run'}
            </button>
          )}
        </div>
      </div>

      <div className={styles.testList}>
        {results.results.map((test, idx) => (
          <TestResultRow key={idx} test={test} />
        ))}
      </div>
    </div>
  )
}

function TestResultRow({ test }: { test: TestResult }) {
  const statusIcon = test.skipped ? '○' : test.passed ? '✓' : '✗'
  const statusClass = test.skipped
    ? styles.skipped
    : test.passed
    ? styles.passed
    : styles.failed

  return (
    <div className={`${styles.testRow} ${statusClass}`}>
      <div className={styles.testHeader}>
        <span className={styles.testIcon}>{statusIcon}</span>
        <span className={styles.testName}>{test.testName}</span>
        <span className={styles.testDuration}>{test.durationMs}ms</span>
      </div>

      {test.error && (
        <div className={styles.errorMessage}>
          {test.error}
        </div>
      )}

      {test.failures && test.failures.length > 0 && (
        <div className={styles.failures}>
          {test.failures.map((f, idx) => (
            <div key={idx} className={styles.failure}>
              <span className={styles.failureType}>{f.type}</span>
              {f.message && <span className={styles.failureMessage}>{f.message}</span>}
              {!f.message && (
                <span className={styles.failureMessage}>
                  expected {f.expected}, got {f.actual}
                </span>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}
