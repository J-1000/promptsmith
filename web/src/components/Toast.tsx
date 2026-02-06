import { useEffect } from 'react'
import styles from './Toast.module.css'

export interface ToastProps {
  message: string
  type?: 'success' | 'error' | 'info'
  onClose: () => void
}

export function Toast({ message, type = 'info', onClose }: ToastProps) {
  useEffect(() => {
    const timer = setTimeout(onClose, 3000)
    return () => clearTimeout(timer)
  }, [onClose])

  return (
    <div className={`${styles.toast} ${styles[type]}`} role="alert">
      <span className={styles.message}>{message}</span>
      <button className={styles.close} onClick={onClose} aria-label="Close">
        &times;
      </button>
    </div>
  )
}
