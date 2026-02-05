import styles from './BenchmarkResults.module.css'

export interface ModelResult {
  model: string
  runs: number
  latencyP50Ms: number
  latencyP99Ms: number
  latencyAvgMs: number
  totalTokensAvg: number
  promptTokens: number
  outputTokensAvg: number
  costPerRequest: number
  totalCost: number
  errors: number
  errorRate: number
}

export interface BenchmarkResult {
  suiteName: string
  promptName: string
  version: string
  models: ModelResult[]
  durationMs: number
  startedAt?: string
  completedAt?: string
}

interface BenchmarkResultsProps {
  results: BenchmarkResult | null
  onRunBenchmark?: () => void
  isRunning?: boolean
}

export function BenchmarkResults({ results, onRunBenchmark, isRunning }: BenchmarkResultsProps) {
  if (!results) {
    return (
      <div className={styles.empty}>
        <div className={styles.emptyIcon}>&#128202;</div>
        <p className={styles.emptyText}>No benchmark results yet</p>
        {onRunBenchmark && (
          <button
            className={styles.runButton}
            onClick={onRunBenchmark}
            disabled={isRunning}
          >
            {isRunning ? 'Running...' : 'Run Benchmark'}
          </button>
        )}
      </div>
    )
  }

  const { models, durationMs } = results
  const bestLatency = getBestByLatency(models)
  const bestCost = getBestByCost(models)

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.headerInfo}>
          <span className={styles.suiteName}>{results.suiteName}</span>
          <span className={styles.version}>v{results.version}</span>
        </div>
        <div className={styles.headerMeta}>
          <span className={styles.duration}>{formatDuration(durationMs)}</span>
          {onRunBenchmark && (
            <button
              className={styles.rerunButton}
              onClick={onRunBenchmark}
              disabled={isRunning}
            >
              {isRunning ? 'Running...' : 'Re-run'}
            </button>
          )}
        </div>
      </div>

      <div className={styles.tableWrapper}>
        <table className={styles.table}>
          <thead>
            <tr>
              <th className={styles.modelCol}>Model</th>
              <th className={styles.numCol}>Latency (p50)</th>
              <th className={styles.numCol}>Latency (p99)</th>
              <th className={styles.numCol}>Tokens</th>
              <th className={styles.numCol}>Cost/Req</th>
              <th className={styles.numCol}>Errors</th>
            </tr>
          </thead>
          <tbody>
            {models.map((model, idx) => (
              <ModelRow
                key={idx}
                model={model}
                isBestLatency={model.model === bestLatency}
                isBestCost={model.model === bestCost}
              />
            ))}
          </tbody>
        </table>
      </div>

      {models.length > 1 && (bestLatency || bestCost) && (
        <div className={styles.recommendation}>
          <span className={styles.recommendIcon}>★</span>
          {bestLatency === bestCost ? (
            <span>{bestLatency} (best latency & cost)</span>
          ) : (
            <span>
              {bestLatency && <span className={styles.recItem}>{bestLatency} for speed</span>}
              {bestLatency && bestCost && <span className={styles.recSeparator}> · </span>}
              {bestCost && <span className={styles.recItem}>{bestCost} for cost</span>}
            </span>
          )}
        </div>
      )}
    </div>
  )
}

interface ModelRowProps {
  model: ModelResult
  isBestLatency: boolean
  isBestCost: boolean
}

function ModelRow({ model, isBestLatency, isBestCost }: ModelRowProps) {
  const hasErrors = model.errorRate >= 1.0

  return (
    <tr className={hasErrors ? styles.errorRow : ''}>
      <td className={styles.modelCol}>
        <span className={styles.modelName}>{model.model}</span>
        {isBestLatency && <span className={styles.badge} title="Fastest">⚡</span>}
        {isBestCost && <span className={styles.badge} title="Cheapest">💰</span>}
      </td>
      <td className={styles.numCol}>
        {hasErrors ? '—' : formatLatency(model.latencyP50Ms)}
      </td>
      <td className={styles.numCol}>
        {hasErrors ? '—' : formatLatency(model.latencyP99Ms)}
      </td>
      <td className={styles.numCol}>
        {hasErrors ? '—' : formatNumber(model.totalTokensAvg)}
      </td>
      <td className={styles.numCol}>
        {hasErrors ? '—' : formatCost(model.costPerRequest)}
      </td>
      <td className={styles.numCol}>
        {model.errors > 0 ? (
          <span className={styles.errorCount}>
            {model.errors} ({formatPercent(model.errorRate)})
          </span>
        ) : (
          <span className={styles.noErrors}>—</span>
        )}
      </td>
    </tr>
  )
}

function getBestByLatency(models: ModelResult[]): string | null {
  const valid = models.filter(m => m.errorRate < 1.0 && m.latencyP50Ms > 0)
  if (valid.length === 0) return null
  return valid.reduce((best, m) => m.latencyP50Ms < best.latencyP50Ms ? m : best).model
}

function getBestByCost(models: ModelResult[]): string | null {
  const valid = models.filter(m => m.errorRate < 1.0 && m.costPerRequest > 0)
  if (valid.length === 0) return null
  return valid.reduce((best, m) => m.costPerRequest < best.costPerRequest ? m : best).model
}

function formatLatency(ms: number): string {
  if (ms < 1000) return `${Math.round(ms)}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function formatNumber(n: number): string {
  return Math.round(n).toLocaleString()
}

function formatCost(cost: number): string {
  if (cost < 0.001) return '<$0.001'
  if (cost < 0.01) return `$${cost.toFixed(4)}`
  return `$${cost.toFixed(3)}`
}

function formatPercent(rate: number): string {
  return `${Math.round(rate * 100)}%`
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`
  return `${Math.floor(ms / 60000)}m ${Math.round((ms % 60000) / 1000)}s`
}
