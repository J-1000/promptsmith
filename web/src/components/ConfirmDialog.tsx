import { useId } from 'react'
import styles from './ConfirmDialog.module.css'

export interface ConfirmDialogProps {
  title: string
  message: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'danger' | 'default'
  onConfirm: () => void
  onCancel: () => void
}

export function ConfirmDialog({
  title,
  message,
  confirmLabel = 'Confirm',
  cancelLabel = 'Cancel',
  variant = 'default',
  onConfirm,
  onCancel,
}: ConfirmDialogProps) {
  const titleId = useId()
  const messageId = useId()

  return (
    <div className={styles.overlay} onClick={onCancel} data-testid="confirm-overlay">
      <div
        className={styles.dialog}
        onClick={(e) => e.stopPropagation()}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        aria-describedby={messageId}
      >
        <h3 id={titleId} className={styles.title}>{title}</h3>
        <p id={messageId} className={styles.message}>{message}</p>
        <div className={styles.actions}>
          <button className={styles.cancelButton} onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            className={`${styles.confirmButton} ${variant === 'danger' ? styles.danger : ''}`}
            onClick={onConfirm}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  )
}
