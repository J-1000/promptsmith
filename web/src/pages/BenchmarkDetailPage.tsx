import { useState, useEffect } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getBenchmark, runBenchmark, BenchmarkSuite } from '../api'
import { BenchmarkResults, BenchmarkResult } from '../components/BenchmarkResults'
import styles from './BenchmarkDetailPage.module.css'

export function BenchmarkDetailPage() {
  const { name } = useParams<{ name: string }>()
  const [benchmark, setBenchmark] = useState<BenchmarkSuite | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [results, setResults] = useState<BenchmarkResult | null>(null)
  const [isRunning, setIsRunning] = useState(false)

  useEffect(() => {
    if (!name) return
    getBenchmark(name)
      .then(setBenchmark)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [name])

  const handleRunBenchmark = async () => {
    if (!name) return
    setIsRunning(true)
    try {
      const result = await runBenchmark(name)
      setResults({
        suiteName: result.suite_name,
        promptName: result.prompt_name,
        version: result.version,
        models: result.models.map((m) => ({
          model: m.model,
          runs: m.runs,
          latencyP50Ms: m.latency_p50_ms,
          latencyP99Ms: m.latency_p99_ms,
          latencyAvgMs: m.latency_p50_ms,
          totalTokensAvg: m.total_tokens_avg,
          promptTokens: 0,
          outputTokensAvg: m.total_tokens_avg,
          costPerRequest: m.cost_per_request,
          totalCost: m.cost_per_request * m.runs,
          errors: m.errors,
          errorRate: m.error_rate,
        })),
        durationMs: result.duration_ms,
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to run benchmark')
    } finally {
      setIsRunning(false)
    }
  }

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading benchmark...</div>
      </div>
    )
  }

  if (error && !benchmark) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load benchmark: {error}</p>
          <Link to="/benchmarks" className={styles.backLink}>Back to benchmarks</Link>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.breadcrumb}>
        <Link to="/benchmarks" className={styles.breadcrumbLink}>Benchmarks</Link>
        <span className={styles.breadcrumbSep}>/</span>
        <span className={styles.breadcrumbCurrent}>{name}</span>
      </div>

      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <h1 className={styles.title}>{name}</h1>
          <span className={styles.badge}>
            {benchmark?.runs_per_model} run{benchmark?.runs_per_model !== 1 ? 's' : ''}/model
          </span>
        </div>
        {benchmark?.prompt && (
          <Link to={`/prompt/${benchmark.prompt}`} className={styles.promptLink}>
            Prompt: <span className={styles.promptName}>{benchmark.prompt}</span>
          </Link>
        )}
      </div>

      {benchmark?.description && (
        <p className={styles.description}>{benchmark.description}</p>
      )}

      <div className={styles.config}>
        <div className={styles.configSection}>
          <h3 className={styles.configTitle}>Models</h3>
          <div className={styles.models}>
            {benchmark?.models.map((model) => (
              <span key={model} className={styles.modelTag}>{model}</span>
            ))}
          </div>
        </div>
        <div className={styles.configSection}>
          <h3 className={styles.configTitle}>Configuration</h3>
          <div className={styles.configMeta}>
            <span>File: <code>{benchmark?.file_path}</code></span>
            <span>Runs per model: <code>{benchmark?.runs_per_model}</code></span>
          </div>
        </div>
      </div>

      <div className={styles.results}>
        <BenchmarkResults
          results={results}
          onRunBenchmark={handleRunBenchmark}
          isRunning={isRunning}
        />
      </div>
    </div>
  )
}
