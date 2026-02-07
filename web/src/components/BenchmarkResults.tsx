import { useCallback } from 'react'
import styles from './BenchmarkResults.module.css'

function downloadBlob(content: string, filename: string, mimeType: string) {
  const blob = new Blob([content], { type: mimeType })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

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

  const exportJSON = useCallback(() => {
    downloadBlob(JSON.stringify(results, null, 2), `${results.suiteName}-benchmark.json`, 'application/json')
  }, [results])

  const exportCSV = useCallback(() => {
    const headers = ['Model', 'Runs', 'Latency P50 (ms)', 'Latency P99 (ms)', 'Latency Avg (ms)', 'Tokens Avg', 'Cost/Request', 'Total Cost', 'Errors', 'Error Rate']
    const rows = models.map(m => [
      m.model, m.runs, m.latencyP50Ms, m.latencyP99Ms, m.latencyAvgMs,
      m.totalTokensAvg, m.costPerRequest, m.totalCost, m.errors, m.errorRate,
    ].join(','))
    downloadBlob([headers.join(','), ...rows].join('\n'), `${results.suiteName}-benchmark.csv`, 'text/csv')
  }, [results, models])

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <div className={styles.headerInfo}>
          <span className={styles.suiteName}>{results.suiteName}</span>
          <span className={styles.version}>v{results.version}</span>
        </div>
        <div className={styles.headerMeta}>
          <span className={styles.duration}>{formatDuration(durationMs)}</span>
          <button className={styles.exportButton} onClick={exportJSON}>JSON</button>
          <button className={styles.exportButton} onClick={exportCSV}>CSV</button>
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

      {models.length > 1 && (
        <RecommendationCards models={models} />
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

function RecommendationCards({ models }: { models: ModelResult[] }) {
  const valid = models.filter(m => m.errorRate < 1.0)
  if (valid.length < 2) return null

  const bestOverall = getBestOverall(valid)
  const bestThroughput = getBestByThroughput(valid)
  const bestBudget = getBestBudget(valid)

  // Deduplicate — only show distinct picks
  const cards: { label: string; model: string; reason: string }[] = []
  if (bestOverall) cards.push({ label: 'Best Overall', model: bestOverall.model, reason: `Balanced score weighing latency, cost, and reliability` })
  if (bestThroughput && (!bestOverall || bestThroughput.model !== bestOverall.model)) {
    cards.push({ label: 'Best Throughput', model: bestThroughput.model, reason: `Lowest avg latency at ${formatLatency(bestThroughput.latencyAvgMs)}` })
  }
  if (bestBudget && (!bestOverall || bestBudget.model !== bestOverall.model) && (!bestThroughput || bestBudget.model !== bestThroughput.model)) {
    cards.push({ label: 'Best Budget', model: bestBudget.model, reason: `Lowest cost at ${formatCost(bestBudget.costPerRequest)}/req` })
  }

  if (cards.length === 0) return null

  return (
    <div className={styles.recCards}>
      {cards.map(card => (
        <div key={card.label} className={styles.recCard}>
          <span className={styles.recCardLabel}>{card.label}</span>
          <span className={styles.recCardModel}>{card.model}</span>
          <span className={styles.recCardReason}>{card.reason}</span>
        </div>
      ))}
    </div>
  )
}

function getBestOverall(models: ModelResult[]): ModelResult | null {
  const valid = models.filter(m => m.errorRate < 1.0 && m.latencyP50Ms > 0 && m.costPerRequest > 0)
  if (valid.length === 0) return null
  // Normalize each metric 0-1 (lower is better) then pick lowest weighted sum
  const maxLat = Math.max(...valid.map(m => m.latencyAvgMs))
  const maxCost = Math.max(...valid.map(m => m.costPerRequest))
  return valid.reduce((best, m) => {
    const score = 0.4 * (m.latencyAvgMs / maxLat) + 0.4 * (m.costPerRequest / maxCost) + 0.2 * m.errorRate
    const bestScore = 0.4 * (best.latencyAvgMs / maxLat) + 0.4 * (best.costPerRequest / maxCost) + 0.2 * best.errorRate
    return score < bestScore ? m : best
  })
}

function getBestByThroughput(models: ModelResult[]): ModelResult | null {
  const valid = models.filter(m => m.errorRate < 1.0 && m.latencyAvgMs > 0)
  if (valid.length === 0) return null
  return valid.reduce((best, m) => m.latencyAvgMs < best.latencyAvgMs ? m : best)
}

function getBestBudget(models: ModelResult[]): ModelResult | null {
  const valid = models.filter(m => m.errorRate < 1.0 && m.costPerRequest > 0)
  if (valid.length === 0) return null
  return valid.reduce((best, m) => m.costPerRequest < best.costPerRequest ? m : best)
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
