import styles from './PageLoader.module.css'

/**
 * Loading fallback shown while a lazily-loaded route chunk is fetched. Gives
 * perceivable feedback instead of a blank frame.
 */
export function PageLoader() {
  return (
    <div className={styles.container} role="status" aria-live="polite">
      <span className={styles.anvil} aria-hidden="true">⚒</span>
      <span className={styles.label}>Heating the forge…</span>
    </div>
  )
}
