import { useState } from 'react'
import styles from './GeneratePanel.module.css'

export type GenerationType = 'variations' | 'compress' | 'expand' | 'rephrase'

export interface Variation {
  content: string
  description: string
  tokenDelta?: number
}

export interface GenerateResult {
  original: string
  variations: Variation[]
  model: string
  type: string
  goal?: string
}

interface GeneratePanelProps {
  onGenerate?: (type: GenerationType, count: number, goal: string) => void
  results: GenerateResult | null
  isGenerating?: boolean
}

const TYPE_DESCRIPTIONS: Record<GenerationType, string> = {
  variations: 'Create alternative versions with different approaches',
  compress: 'Reduce token count while preserving functionality',
  expand: 'Add more detail, examples, and edge cases',
  rephrase: 'Reword while keeping the same meaning',
}

export function GeneratePanel({ onGenerate, results, isGenerating }: GeneratePanelProps) {
  const [type, setType] = useState<GenerationType>('variations')
  const [count, setCount] = useState(3)
  const [goal, setGoal] = useState('')
  const [expandedIdx, setExpandedIdx] = useState<number | null>(null)

  const handleGenerate = () => {
    onGenerate?.(type, count, goal)
  }

  return (
    <div className={styles.container}>
      <div className={styles.controls}>
        <div className={styles.controlGroup}>
          <label className={styles.label}>Type</label>
          <div className={styles.typeButtons}>
            {(Object.keys(TYPE_DESCRIPTIONS) as GenerationType[]).map((t) => (
              <button
                key={t}
                className={`${styles.typeButton} ${type === t ? styles.typeButtonActive : ''}`}
                onClick={() => setType(t)}
                title={TYPE_DESCRIPTIONS[t]}
              >
                {t}
              </button>
            ))}
          </div>
        </div>

        <div className={styles.controlRow}>
          <div className={styles.controlGroup}>
            <label className={styles.label}>Count</label>
            <select
              className={styles.select}
              value={count}
              onChange={(e) => setCount(parseInt(e.target.value))}
            >
              {[1, 2, 3, 4, 5].map((n) => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
          </div>

          <div className={styles.controlGroup} style={{ flex: 1 }}>
            <label className={styles.label}>Goal (optional)</label>
            <input
              type="text"
              className={styles.input}
              placeholder="e.g., be more concise, improve clarity"
              value={goal}
              onChange={(e) => setGoal(e.target.value)}
            />
          </div>
        </div>

        <button
          className={styles.generateButton}
          onClick={handleGenerate}
          disabled={isGenerating}
        >
          {isGenerating ? 'Generating...' : 'Generate'}
        </button>
      </div>

      {results && results.variations.length > 0 && (
        <div className={styles.results}>
          <div className={styles.resultsHeader}>
            <span className={styles.resultsTitle}>
              {results.variations.length} {results.type} generated
            </span>
            <span className={styles.resultsModel}>using {results.model}</span>
          </div>

          <div className={styles.variationsList}>
            {results.variations.map((v, idx) => (
              <div
                key={idx}
                className={`${styles.variation} ${expandedIdx === idx ? styles.variationExpanded : ''}`}
              >
                <div
                  className={styles.variationHeader}
                  onClick={() => setExpandedIdx(expandedIdx === idx ? null : idx)}
                >
                  <span className={styles.variationNumber}>#{idx + 1}</span>
                  <span className={styles.variationDesc}>{v.description || 'Variation'}</span>
                  {v.tokenDelta !== undefined && v.tokenDelta !== 0 && (
                    <span className={v.tokenDelta < 0 ? styles.tokensSaved : styles.tokensAdded}>
                      {v.tokenDelta > 0 ? '+' : ''}{v.tokenDelta} tokens
                    </span>
                  )}
                  <span className={styles.expandIcon}>{expandedIdx === idx ? '−' : '+'}</span>
                </div>

                {expandedIdx === idx && (
                  <div className={styles.variationContent}>
                    <pre className={styles.variationCode}>{v.content}</pre>
                    <button
                      className={styles.copyButton}
                      onClick={() => navigator.clipboard.writeText(v.content)}
                    >
                      Copy
                    </button>
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      {!results && !isGenerating && (
        <div className={styles.empty}>
          <div className={styles.emptyIcon}>&#9889;</div>
          <p className={styles.emptyText}>Generate variations of your prompt using AI</p>
          <p className={styles.emptyHint}>
            Choose a generation type and click Generate to create alternatives
          </p>
        </div>
      )}
    </div>
  )
}
