import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { listChains, createChain, Chain } from '../api'
import styles from './ChainsPage.module.css'

export function ChainsPage() {
  const [chains, setChains] = useState<Chain[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [search, setSearch] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [newName, setNewName] = useState('')
  const [newDesc, setNewDesc] = useState('')
  const [creating, setCreating] = useState(false)

  useEffect(() => {
    listChains()
      .then(setChains)
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [])

  const handleCreate = async () => {
    if (!newName.trim()) return
    setCreating(true)
    try {
      await createChain(newName.trim(), newDesc.trim())
      const updated = await listChains()
      setChains(updated)
      setShowCreate(false)
      setNewName('')
      setNewDesc('')
    } catch (err: any) {
      setError(err.message)
    } finally {
      setCreating(false)
    }
  }

  const filtered = chains.filter((c) =>
    c.name.toLowerCase().includes(search.toLowerCase()) ||
    c.description.toLowerCase().includes(search.toLowerCase())
  )

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading chains...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load chains: {error}</p>
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
            <h1 className={styles.title}>Prompt Chains</h1>
            <p className={styles.subtitle}>
              {chains.length} chain{chains.length !== 1 ? 's' : ''} configured
            </p>
          </div>
          <div className={styles.actions}>
            <button
              className={styles.createButton}
              onClick={() => setShowCreate(true)}
            >
              + New Chain
            </button>
          </div>
        </div>
        {chains.length > 0 && (
          <input
            type="text"
            className={styles.searchInput}
            placeholder="Search chains..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        )}
      </div>

      {chains.length === 0 ? (
        <div className={styles.empty}>
          <p>No chains found.</p>
          <p className={styles.hint}>Create a chain to sequence prompts into a pipeline.</p>
        </div>
      ) : filtered.length === 0 ? (
        <div className={styles.empty}>
          <p>No chains matching "{search}"</p>
        </div>
      ) : (
        <div className={styles.list}>
          {filtered.map((chain) => (
            <Link
              key={chain.id}
              to={`/chains/${chain.name}`}
              className={styles.card}
            >
              <div className={styles.cardHeader}>
                <span className={styles.chainName}>{chain.name}</span>
                <span className={styles.stepCount}>
                  {chain.step_count} step{chain.step_count !== 1 ? 's' : ''}
                </span>
              </div>
              {chain.description && (
                <p className={styles.description}>{chain.description}</p>
              )}
              <div className={styles.cardFooter}>
                <span className={styles.timestamp}>
                  Updated {new Date(chain.updated_at).toLocaleDateString()}
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}

      {showCreate && (
        <div className={styles.modalOverlay} onClick={() => setShowCreate(false)}>
          <div className={styles.modal} onClick={(e) => e.stopPropagation()}>
            <h2 className={styles.modalTitle}>Create Chain</h2>
            <div className={styles.modalField}>
              <label className={styles.modalLabel}>Name</label>
              <input
                className={styles.modalInput}
                placeholder="e.g. summarize-translate"
                value={newName}
                onChange={(e) => setNewName(e.target.value)}
                autoFocus
              />
            </div>
            <div className={styles.modalField}>
              <label className={styles.modalLabel}>Description</label>
              <input
                className={styles.modalInput}
                placeholder="What does this chain do?"
                value={newDesc}
                onChange={(e) => setNewDesc(e.target.value)}
              />
            </div>
            <div className={styles.modalActions}>
              <button className={styles.modalCancel} onClick={() => setShowCreate(false)}>
                Cancel
              </button>
              <button
                className={styles.modalSubmit}
                onClick={handleCreate}
                disabled={!newName.trim() || creating}
              >
                {creating ? 'Creating...' : 'Create'}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
