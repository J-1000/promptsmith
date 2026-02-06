import { Outlet, Link, useLocation } from 'react-router-dom'
import styles from './Layout.module.css'

export function Layout() {
  const location = useLocation()
  const path = location.pathname

  const isActive = (route: string) => {
    if (route === '/') return path === '/' || path.startsWith('/prompt/')
    return path.startsWith(route)
  }

  return (
    <div className={styles.layout}>
      <header className={styles.header}>
        <Link to="/" className={styles.logo}>
          <span className={styles.logoIcon}>&#9881;</span>
          <span className={styles.logoText}>PromptSmith</span>
        </Link>
        <nav className={styles.nav}>
          <Link to="/" className={`${styles.navLink} ${isActive('/') ? styles.navLinkActive : ''}`}>Prompts</Link>
          <Link to="/tests" className={`${styles.navLink} ${isActive('/tests') ? styles.navLinkActive : ''}`}>Tests</Link>
          <Link to="/benchmarks" className={`${styles.navLink} ${isActive('/benchmarks') ? styles.navLinkActive : ''}`}>Benchmarks</Link>
          <Link to="/settings" className={`${styles.navLink} ${isActive('/settings') ? styles.navLinkActive : ''}`}>Settings</Link>
        </nav>
      </header>
      <main className={styles.main}>
        <Outlet />
      </main>
    </div>
  )
}
