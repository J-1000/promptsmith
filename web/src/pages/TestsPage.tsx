import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { listTests, TestSuite } from '../api'
import styles from './TestsPage.module.css'

export function TestsPage() {
  const [suites, setSuites] = useState<TestSuite[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  useEffect(() => {
    listTests()
      .then(setSuites)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const handleRefresh = () => {
    setLoading(true)
    setError(null)
    listTests()
      .then(setSuites)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  const filtered = suites.filter((s) =>
    s.name.toLowerCase().includes(search.toLowerCase()) ||
    s.prompt.toLowerCase().includes(search.toLowerCase())
  )

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading test suites...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load tests: {error}</p>
          <p className={styles.hint}>Make sure the server is running: <code>promptsmith serve</code></p>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.headerRow}>
          <div className={styles.headerTop}>
            <h1 className={styles.title}>Test Suites</h1>
            <p className={styles.subtitle}>
              {suites.length} test suite{suites.length !== 1 ? 's' : ''} configured
            </p>
          </div>
          <button
            className={styles.refreshButton}
            onClick={handleRefresh}
            disabled={loading}
            title="Refresh test suites"
          >
            {loading ? '...' : '↻'}
          </button>
        </div>
        {suites.length > 0 && (
          <input
            type="text"
            className={styles.searchInput}
            placeholder="Search test suites..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        )}
      </div>

      {suites.length === 0 ? (
        <div className={styles.empty}>
          <p>No test suites found.</p>
          <p className={styles.hint}>Create a test suite in <code>tests/*.test.yaml</code></p>
        </div>
      ) : filtered.length === 0 ? (
        <div className={styles.empty}>
          <p>No test suites matching "{search}"</p>
        </div>
      ) : (
        <div className={styles.list}>
          {filtered.map((suite) => (
            <Link
              key={suite.name}
              to={`/tests/${suite.name}`}
              className={styles.card}
            >
              <div className={styles.cardHeader}>
                <span className={styles.suiteName}>{suite.name}</span>
                <span className={styles.testCount}>{suite.test_count} test{suite.test_count !== 1 ? 's' : ''}</span>
              </div>
              {suite.description && (
                <p className={styles.description}>{suite.description}</p>
              )}
              <div className={styles.cardFooter}>
                <span className={styles.promptLink}>
                  Prompt: <span className={styles.promptName}>{suite.prompt}</span>
                </span>
                <span className={styles.filePath}>{suite.file_path}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
