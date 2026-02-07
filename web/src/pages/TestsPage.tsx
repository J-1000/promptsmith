import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { listTests, listTestRuns, TestSuite } from '../api'
import styles from './TestsPage.module.css'

interface FlakinessMap {
  [suiteName: string]: number // percentage 0-100
}

export function TestsPage() {
  const [suites, setSuites] = useState<TestSuite[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [flakiness, setFlakiness] = useState<FlakinessMap>({})

  useEffect(() => {
    listTests()
      .then((loadedSuites) => {
        setSuites(loadedSuites)
        // Fetch run history for each suite to compute flakiness
        loadedSuites.forEach((suite) => {
          listTestRuns(suite.name).then((runs) => {
            if (runs.length >= 2) {
              const statuses = runs.map(r => r.status)
              const transitions = statuses.slice(1).filter((s, i) => s !== statuses[i]).length
              const flakyPct = Math.round((transitions / (statuses.length - 1)) * 100)
              setFlakiness(prev => ({ ...prev, [suite.name]: flakyPct }))
            }
          }).catch(() => { /* ignore errors fetching runs */ })
        })
      })
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
                <div className={styles.cardHeaderLeft}>
                  <span className={styles.suiteName}>{suite.name}</span>
                  {flakiness[suite.name] !== undefined && flakiness[suite.name] > 20 && (
                    <span className={styles.flakyBadge} title={`${flakiness[suite.name]}% flakiness rate`}>
                      Flaky {flakiness[suite.name]}%
                    </span>
                  )}
                </div>
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
