import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { listBenchmarks, BenchmarkSuite } from '../api'
import styles from './BenchmarksPage.module.css'

export function BenchmarksPage() {
  const [benchmarks, setBenchmarks] = useState<BenchmarkSuite[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  useEffect(() => {
    listBenchmarks()
      .then(setBenchmarks)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const handleRefresh = () => {
    setLoading(true)
    setError(null)
    listBenchmarks()
      .then(setBenchmarks)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  const filtered = benchmarks.filter((b) =>
    b.name.toLowerCase().includes(search.toLowerCase()) ||
    b.prompt.toLowerCase().includes(search.toLowerCase())
  )

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading benchmarks...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load benchmarks: {error}</p>
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
            <h1 className={styles.title}>Benchmarks</h1>
            <p className={styles.subtitle}>
              {benchmarks.length} benchmark{benchmarks.length !== 1 ? 's' : ''} configured
            </p>
          </div>
          <button
            className={styles.refreshButton}
            onClick={handleRefresh}
            disabled={loading}
            title="Refresh benchmarks"
          >
            {loading ? '...' : '↻'}
          </button>
        </div>
        {benchmarks.length > 0 && (
          <input
            type="text"
            className={styles.searchInput}
            placeholder="Search benchmarks..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        )}
      </div>

      {benchmarks.length === 0 ? (
        <div className={styles.empty}>
          <p>No benchmarks found.</p>
          <p className={styles.hint}>Create a benchmark in <code>benchmarks/*.bench.yaml</code></p>
        </div>
      ) : filtered.length === 0 ? (
        <div className={styles.empty}>
          <p>No benchmarks matching "{search}"</p>
        </div>
      ) : (
        <div className={styles.list}>
          {filtered.map((bench) => (
            <Link
              key={bench.name}
              to={`/benchmarks/${bench.name}`}
              className={styles.card}
            >
              <div className={styles.cardHeader}>
                <span className={styles.benchName}>{bench.name}</span>
                <span className={styles.runsBadge}>
                  {bench.runs_per_model} run{bench.runs_per_model !== 1 ? 's' : ''}/model
                </span>
              </div>
              {bench.description && (
                <p className={styles.description}>{bench.description}</p>
              )}
              <div className={styles.models}>
                {bench.models.map((model) => (
                  <span key={model} className={styles.modelTag}>{model}</span>
                ))}
              </div>
              <div className={styles.cardFooter}>
                <span className={styles.promptLink}>
                  Prompt: <span className={styles.promptName}>{bench.prompt}</span>
                </span>
                <span className={styles.filePath}>{bench.file_path}</span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
