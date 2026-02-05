import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { listPrompts, Prompt } from '../api'
import styles from './HomePage.module.css'

export function HomePage() {
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')

  useEffect(() => {
    listPrompts()
      .then(setPrompts)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const filteredPrompts = prompts.filter((p) =>
    p.name.toLowerCase().includes(search.toLowerCase()) ||
    p.description?.toLowerCase().includes(search.toLowerCase())
  )

  const handleRefresh = () => {
    setLoading(true)
    setError(null)
    listPrompts()
      .then(setPrompts)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
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
      <div className={styles.header}>
        <div className={styles.headerRow}>
          <div className={styles.headerTop}>
            <h1 className={styles.title}>Prompts</h1>
            <p className={styles.subtitle}>
              {prompts.length} prompts tracked
            </p>
          </div>
          <button
            className={styles.refreshButton}
            onClick={handleRefresh}
            disabled={loading}
            title="Refresh prompts"
          >
            {loading ? '...' : '↻'}
          </button>
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
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}
