import { Link } from 'react-router-dom'
import styles from './HomePage.module.css'

// Mock data - will be replaced with CLI/API integration
const mockPrompts = [
  {
    name: 'greeting',
    description: 'A friendly greeting prompt',
    version: '1.0.2',
    lastModified: '2024-01-15',
    tags: ['prod'],
  },
  {
    name: 'summarize',
    description: 'Summarizes long text into key points',
    version: '2.1.0',
    lastModified: '2024-01-14',
    tags: ['staging', 'prod'],
  },
  {
    name: 'code-review',
    description: 'Reviews code and suggests improvements',
    version: '1.0.0',
    lastModified: '2024-01-10',
    tags: [],
  },
]

export function HomePage() {
  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Prompts</h1>
        <p className={styles.subtitle}>
          {mockPrompts.length} prompts tracked
        </p>
      </div>

      <div className={styles.grid}>
        {mockPrompts.map((prompt) => (
          <Link
            key={prompt.name}
            to={`/prompt/${prompt.name}`}
            className={styles.card}
          >
            <div className={styles.cardHeader}>
              <span className={styles.promptName}>{prompt.name}</span>
              <span className={styles.version}>v{prompt.version}</span>
            </div>
            <p className={styles.description}>{prompt.description}</p>
            <div className={styles.cardFooter}>
              <span className={styles.date}>{prompt.lastModified}</span>
              {prompt.tags.length > 0 && (
                <div className={styles.tags}>
                  {prompt.tags.map((tag) => (
                    <span key={tag} className={styles.tag}>{tag}</span>
                  ))}
                </div>
              )}
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}
