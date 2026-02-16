import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { listPrompts, listTests, listBenchmarks, createPrompt, getDashboardActivity, getDashboardHealth, Prompt, TestSuite, BenchmarkSuite, ActivityEvent, PromptHealth } from '../api'
import { Toast } from '../components/Toast'
import styles from './HomePage.module.css'

export function HomePage() {
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [tests, setTests] = useState<TestSuite[]>([])
  const [benchmarks, setBenchmarks] = useState<BenchmarkSuite[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [showNewModal, setShowNewModal] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDescription, setNewDescription] = useState('')
  const [newContent, setNewContent] = useState('')
  const [creating, setCreating] = useState(false)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' | 'info' } | null>(null)
  const [activity, setActivity] = useState<ActivityEvent[]>([])
  const [healthMap, setHealthMap] = useState<Record<string, PromptHealth>>({})

  useEffect(() => {
    Promise.all([
      listPrompts(),
      listTests().catch(() => [] as TestSuite[]),
      listBenchmarks().catch(() => [] as BenchmarkSuite[]),
      getDashboardActivity(10).catch(() => [] as ActivityEvent[]),
      getDashboardHealth().catch(() => [] as PromptHealth[]),
    ])
      .then(([promptsData, testsData, benchmarksData, activityData, healthData]) => {
        setPrompts(promptsData)
        setTests(testsData)
        setBenchmarks(benchmarksData)
        setActivity(activityData)
        const map: Record<string, PromptHealth> = {}
        for (const h of healthData) {
          map[h.prompt_name] = h
        }
        setHealthMap(map)
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const filteredPrompts = prompts.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.description?.toLowerCase().includes(search.toLowerCase())
  )

  const totalTests = tests.reduce((sum, s) => sum + s.test_count, 0)

  const refreshData = () => {
    return Promise.all([
      listPrompts(),
      listTests().catch(() => [] as TestSuite[]),
      listBenchmarks().catch(() => [] as BenchmarkSuite[]),
      getDashboardActivity(10).catch(() => [] as ActivityEvent[]),
      getDashboardHealth().catch(() => [] as PromptHealth[]),
    ]).then(([promptsData, testsData, benchmarksData, activityData, healthData]) => {
      setPrompts(promptsData)
      setTests(testsData)
      setBenchmarks(benchmarksData)
      setActivity(activityData)
      const map: Record<string, PromptHealth> = {}
      for (const h of healthData) {
        map[h.prompt_name] = h
      }
      setHealthMap(map)
    })
  }

  const handleRefresh = () => {
    setLoading(true)
    setError(null)
    refreshData()
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }

  const handleCreatePrompt = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!newName.trim()) return
    setCreating(true)
    try {
      await createPrompt(newName.trim(), newDescription.trim(), newContent.trim() || undefined)
      setShowNewModal(false)
      setNewName('')
      setNewDescription('')
      setNewContent('')
      await refreshData()
      setToast({ message: `Prompt "${newName.trim()}" created`, type: 'success' })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to create prompt'
      setToast({ message: msg, type: 'error' })
    } finally {
      setCreating(false)
    }
  }

  const formatRelativeTime = (timestamp: string) => {
    const diff = Date.now() - new Date(timestamp).getTime()
    const minutes = Math.floor(diff / 60000)
    if (minutes < 1) return 'just now'
    if (minutes < 60) return `${minutes}m ago`
    const hours = Math.floor(minutes / 60)
    if (hours < 24) return `${hours}h ago`
    const days = Math.floor(hours / 24)
    return `${days}d ago`
  }

  const activityIcon = (type: string) => {
    switch (type) {
      case 'version': return '<>'
      case 'test_run': return '\u2713'
      case 'benchmark_run': return '\u25A0'
      default: return '\u2022'
    }
  }

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading prompts...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load prompts: {error}</p>
          <p className={styles.hint}>Make sure the server is running: <code>promptsmith serve</code></p>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.stats}>
        <Link to="/" className={styles.statCard}>
          <span className={styles.statValue}>{prompts.length}</span>
          <span className={styles.statLabel}>Prompts</span>
        </Link>
        <Link to="/tests" className={styles.statCard}>
          <span className={styles.statValue}>{tests.length}</span>
          <span className={styles.statLabel}>Test Suites</span>
        </Link>
        <Link to="/tests" className={styles.statCard}>
          <span className={styles.statValue}>{totalTests}</span>
          <span className={styles.statLabel}>Test Cases</span>
        </Link>
        <Link to="/benchmarks" className={styles.statCard}>
          <span className={styles.statValue}>{benchmarks.length}</span>
          <span className={styles.statLabel}>Benchmarks</span>
        </Link>
      </div>

      {activity.length > 0 && (
        <div className={styles.activitySection}>
          <h2 className={styles.activityTitle}>Recent Activity</h2>
          <div className={styles.activityList}>
            {activity.map((event, i) => (
              <div key={i} className={styles.activityItem}>
                <span className={`${styles.activityIcon} ${styles[`activityIcon_${event.type}`] || ''}`}>
                  {activityIcon(event.type)}
                </span>
                <div className={styles.activityContent}>
                  <span className={styles.activityItemTitle}>
                    {event.title}
                    <span className={styles.activityPrompt}>{event.prompt_name}</span>
                  </span>
                  <span className={styles.activityDetail}>{event.detail}</span>
                </div>
                <span className={styles.activityTime}>{formatRelativeTime(event.timestamp)}</span>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className={styles.header}>
        <div className={styles.headerRow}>
          <div className={styles.headerTop}>
            <h1 className={styles.title}>Prompts</h1>
            <p className={styles.subtitle}>
              {prompts.length} prompts tracked
            </p>
          </div>
          <div className={styles.headerActions}>
            <button
              className={styles.newButton}
              onClick={() => setShowNewModal(true)}
            >
              + New Prompt
            </button>
            <button
              className={styles.refreshButton}
              onClick={handleRefresh}
              disabled={loading}
              title="Refresh prompts"
            >
              {loading ? '...' : '↻'}
            </button>
          </div>
        </div>
        {prompts.length > 0 && (
          <input
            type="text"
            className={styles.searchInput}
            placeholder="Search prompts..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        )}
      </div>

      {prompts.length === 0 ? (
        <div className={styles.empty}>
          <p>No prompts tracked yet.</p>
          <p className={styles.hint}>Add a prompt with: <code>promptsmith add &lt;file&gt;</code></p>
        </div>
      ) : filteredPrompts.length === 0 ? (
        <div className={styles.empty}>
          <p>No prompts matching "{search}"</p>
        </div>
      ) : (
        <div className={styles.grid}>
          {filteredPrompts.map((prompt) => (
            <Link
              key={prompt.name}
              to={`/prompt/${prompt.name}`}
              className={styles.card}
            >
              <div className={styles.cardHeader}>
                <span className={styles.promptName}>{prompt.name}</span>
                {prompt.version && <span className={styles.version}>v{prompt.version}</span>}
              </div>
              <p className={styles.description}>{prompt.description || 'No description'}</p>
              <div className={styles.cardFooter}>
                <span className={styles.date}>{new Date(prompt.created_at).toLocaleDateString()}</span>
                {healthMap[prompt.name] && (
                  <div className={styles.healthBadge}>
                    <span className={`${styles.healthDot} ${styles[`healthDot_${healthMap[prompt.name].last_test_status}`] || ''}`} />
                    {healthMap[prompt.name].last_test_status !== 'none' && (
                      <span className={styles.healthText}>
                        {Math.round(healthMap[prompt.name].test_pass_rate * 100)}% passing
                      </span>
                    )}
                  </div>
                )}
              </div>
            </Link>
          ))}
        </div>
      )}

      {showNewModal && (
        <div className={styles.modalOverlay} onClick={() => setShowNewModal(false)}>
          <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.modalTitle}>New Prompt</h2>
            <form onSubmit={handleCreatePrompt} className={styles.modalForm}>
              <label className={styles.label}>
                Name
                <input
                  type="text"
                  className={styles.input}
                  value={newName}
                  onChange={(e) => setNewName(e.target.value)}
                  placeholder="my-prompt"
                  autoFocus
                />
              </label>
              <label className={styles.label}>
                Description
                <input
                  type="text"
                  className={styles.input}
                  value={newDescription}
                  onChange={(e) => setNewDescription(e.target.value)}
                  placeholder="What does this prompt do?"
                />
              </label>
              <label className={styles.label}>
                Content (optional)
                <textarea
                  className={styles.textarea}
                  value={newContent}
                  onChange={(e) => setNewContent(e.target.value)}
                  placeholder="You are a helpful assistant..."
                  rows={4}
                />
              </label>
              <div className={styles.modalActions}>
                <button type="button" className={styles.cancelButton} onClick={() => setShowNewModal(false)}>
                  Cancel
                </button>
                <button type="submit" className={styles.submitButton} disabled={creating || !newName.trim()}>
                  {creating ? 'Creating...' : 'Create'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}
    </div>
  )
}
