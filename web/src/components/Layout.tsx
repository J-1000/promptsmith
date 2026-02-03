import { Outlet, Link } from 'react-router-dom'
import styles from './Layout.module.css'

export function Layout() {
  return (
    <div className={styles.layout}>
      <header className={styles.header}>
        <Link to="/" className={styles.logo}>
          <span className={styles.logoIcon}>&#9881;</span>
          <span className={styles.logoText}>PromptSmith</span>
        </Link>
        <nav className={styles.nav}>
          <Link to="/" className={styles.navLink}>Prompts</Link>
        </nav>
      </header>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  )
}
