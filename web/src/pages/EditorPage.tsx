import { useState, useEffect, useMemo } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getPrompt, getPromptVersions, createVersion, Prompt } from '../api'
import styles from './EditorPage.module.css'

function extractVariables(content: string): string[] {
  const vars: string[] = []
  const seen = new Set<string>()
  const regex = /\{\{(\s*\w+\s*)\}\}/g
  let match
  while ((match = regex.exec(content)) !== null) {
    const name = match[1].trim()
    if (!seen.has(name)) {
      vars.push(name)
      seen.add(name)
    }
  }
  return vars
}

function estimateTokens(text: string): number {
  // Rough estimation: ~4 characters per token for English text
  return Math.ceil(text.length / 4)
}

export function EditorPage() {
  const { name } = useParams<{ name: string }>()
  const [prompt, setPrompt] = useState<Prompt | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [content, setContent] = useState('')
  const [originalContent, setOriginalContent] = useState('')
  const [commitMessage, setCommitMessage] = useState('')
  const [saving, setSaving] = useState(false)
  const [saveSuccess, setSaveSuccess] = useState(false)
  const [currentVersion, setCurrentVersion] = useState('')

  useEffect(() => {
    if (!name) return
    Promise.all([getPrompt(name), getPromptVersions(name)])
      .then(([promptData, versions]) => {
        setPrompt(promptData)
        if (versions.length > 0) {
          setContent(versions[0].content)
          setOriginalContent(versions[0].content)
          setCurrentVersion(versions[0].version)
        }
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [name])

  const variables = useMemo(() => extractVariables(content), [content])
  const tokens = useMemo(() => estimateTokens(content), [content])
  const hasChanges = content !== originalContent

  const handleSave = async () => {
    if (!name || !hasChanges) return
    setSaving(true)
    setSaveSuccess(false)
    try {
      const msg = commitMessage || 'Updated via web editor'
      const version = await createVersion(name, content, msg)
      setOriginalContent(content)
      setCurrentVersion(version.version)
      setCommitMessage('')
      setSaveSuccess(true)
      setTimeout(() => setSaveSuccess(false), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading editor...</div>
      </div>
    )
  }

  if (error && !prompt) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load prompt: {error}</p>
          <Link to="/" className={styles.backLink}>Back to prompts</Link>
        </div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.breadcrumb}>
        <Link to="/" className={styles.breadcrumbLink}>Prompts</Link>
        <span className={styles.breadcrumbSep}>/</span>
        <Link to={`/prompt/${name}`} className={styles.breadcrumbLink}>{name}</Link>
        <span className={styles.breadcrumbSep}>/</span>
        <span className={styles.breadcrumbCurrent}>Edit</span>
      </div>

      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <h1 className={styles.title}>Edit {name}</h1>
          {currentVersion && <span className={styles.version}>v{currentVersion}</span>}
          {hasChanges && <span className={styles.unsaved}>Unsaved changes</span>}
        </div>
        <div className={styles.headerRight}>
          <Link to={`/prompt/${name}`} className={styles.cancelButton}>Cancel</Link>
        </div>
      </div>

      <div className={styles.editorLayout}>
        <div className={styles.editorMain}>
          <div className={styles.editorHeader}>
            <span className={styles.fileName}>{prompt?.file_path || `${name}.prompt`}</span>
            <div className={styles.editorMeta}>
              <span className={styles.tokenCount}>{tokens} tokens (est.)</span>
              <span className={styles.charCount}>{content.length} chars</span>
              <span className={styles.lineCount}>{content.split('\n').length} lines</span>
            </div>
          </div>
          <textarea
            className={styles.editor}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            spellCheck={false}
            placeholder="Enter your prompt content here..."
          />
        </div>

        <div className={styles.sidebar}>
          <div className={styles.sidebarSection}>
            <h3 className={styles.sidebarTitle}>Variables</h3>
            {variables.length === 0 ? (
              <p className={styles.sidebarHint}>
                No variables detected. Use <code>{'{{varName}}'}</code> syntax.
              </p>
            ) : (
              <div className={styles.variableList}>
                {variables.map((v) => (
                  <div key={v} className={styles.variable}>
                    <span className={styles.variableIcon}>{'{ }'}</span>
                    <span className={styles.variableName}>{v}</span>
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className={styles.sidebarSection}>
            <h3 className={styles.sidebarTitle}>Save as New Version</h3>
            <input
              type="text"
              className={styles.commitInput}
              placeholder="Commit message (optional)"
              value={commitMessage}
              onChange={(e) => setCommitMessage(e.target.value)}
            />
            <button
              className={styles.saveButton}
              onClick={handleSave}
              disabled={!hasChanges || saving}
            >
              {saving ? 'Saving...' : saveSuccess ? 'Saved!' : 'Save Version'}
            </button>
            {error && <p className={styles.saveError}>{error}</p>}
          </div>
        </div>
      </div>
    </div>
  )
}
