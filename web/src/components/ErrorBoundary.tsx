import { Component, type ErrorInfo, type ReactNode } from 'react'
import styles from './ErrorBoundary.module.css'

interface Props {
  children: ReactNode
}

interface State {
  error: Error | null
}

/**
 * Catches render-time errors in the route tree so a single failing page does
 * not unmount the whole app, and offers a path back to a working state.
 */
export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null }

  static getDerivedStateFromError(error: Error): State {
    return { error }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    // Surface the failure for diagnostics; in production this is where a
    // reporting hook (Sentry, etc.) would live.
    console.error('Unhandled UI error:', error, info.componentStack)
  }

  handleReset = () => {
    this.setState({ error: null })
  }

  render() {
    const { error } = this.state
    if (!error) {
      return this.props.children
    }

    return (
      <div className={styles.container} role="alert">
        <div className={styles.panel}>
          <span className={styles.spark}>⚡</span>
          <h1 className={styles.title}>The forge sputtered</h1>
          <p className={styles.message}>
            Something broke while rendering this view. Your data is safe — try again or reload.
          </p>
          {error.message && <pre className={styles.detail}>{error.message}</pre>}
          <div className={styles.actions}>
            <button className={styles.primary} onClick={this.handleReset}>
              Try again
            </button>
            <button className={styles.secondary} onClick={() => window.location.reload()}>
              Reload page
            </button>
          </div>
        </div>
      </div>
    )
  }
}
