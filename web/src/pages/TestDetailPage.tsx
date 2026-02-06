import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getTest, runTest, TestSuite } from '../api'
import { TestResults, SuiteResult } from '../components/TestResults'
import styles from './TestDetailPage.module.css'

export function TestDetailPage() {
  const { name } = useParams<{ name: string }>()
  const [suite, setSuite] = useState<TestSuite | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [testResults, setTestResults] = useState<SuiteResult | null>(null)
  const [isRunning, setIsRunning] = useState(false)

  useEffect(() => {
    if (!name) return
    getTest(name)
      .then(setSuite)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [name])

  const handleRunTests = async () => {
    if (!name) return
    setIsRunning(true)
    try {
      const result = await runTest(name)
      setTestResults({
        suiteName: result.suite_name,
        promptName: result.prompt_name,
        version: result.version,
        passed: result.passed,
        failed: result.failed,
        skipped: result.skipped,
        total: result.total,
        durationMs: result.duration_ms,
        results: result.results.map((r) => ({
          testName: r.test_name,
          passed: r.passed,
          skipped: r.skipped,
          output: r.output,
          failures: r.failures?.map((f) => ({
            type: f.type,
            message: f.message,
            expected: f.expected,
            actual: f.actual,
          })),
          error: r.error,
          durationMs: r.duration_ms,
        })),
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to run tests')
    } finally {
      setIsRunning(false)
    }
  }

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading test suite...</div>
      </div>
    )
  }

  if (error && !suite) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load test suite: {error}</p>
          <Link to="/tests" className={styles.backLink}>Back to tests</Link>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.breadcrumb}>
        <Link to="/tests" className={styles.breadcrumbLink}>Tests</Link>
        <span className={styles.breadcrumbSep}>/</span>
        <span className={styles.breadcrumbCurrent}>{name}</span>
      </div>

      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <h1 className={styles.title}>{name}</h1>
          <span className={styles.badge}>{suite?.test_count} test{suite?.test_count !== 1 ? 's' : ''}</span>
        </div>
        {suite?.prompt && (
          <Link to={`/prompt/${suite.prompt}`} className={styles.promptLink}>
            Prompt: <span className={styles.promptName}>{suite.prompt}</span>
          </Link>
        )}
      </div>

      {suite?.description && (
        <p className={styles.description}>{suite.description}</p>
      )}

      <div className={styles.meta}>
        <span className={styles.metaItem}>
          File: <code>{suite?.file_path}</code>
        </span>
      </div>

      <div className={styles.results}>
        <TestResults
          results={testResults}
          onRunTests={handleRunTests}
          isRunning={isRunning}
        />
      </div>
    </div>
  )
}
