import { useState, useEffect, useMemo, useRef, useCallback } from 'react'
import { useParams, Link } from 'react-router-dom'
import { getPrompt, getPromptVersions, createVersion, Prompt } from '../api'
import { EditorView, keymap, ViewUpdate, Decoration, DecorationSet, ViewPlugin } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { markdown } from '@codemirror/lang-markdown'
import { syntaxHighlighting, HighlightStyle } from '@codemirror/language'
import { tags } from '@lezer/highlight'
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
  return Math.ceil(text.length / 4)
}

// Custom theme matching the dark forge aesthetic
const forgeTheme = EditorView.theme({
  '&': {
    backgroundColor: 'var(--bg-secondary)',
    color: 'var(--text-primary)',
    fontFamily: 'var(--font-mono)',
    fontSize: '13px',
  },
  '.cm-content': {
    padding: '16px',
    lineHeight: '1.6',
    caretColor: 'var(--accent-primary)',
  },
  '&.cm-focused .cm-cursor': {
    borderLeftColor: 'var(--accent-primary)',
  },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
    backgroundColor: 'rgba(245, 166, 35, 0.15)',
  },
  '.cm-activeLine': {
    backgroundColor: 'rgba(245, 166, 35, 0.05)',
  },
  '.cm-gutters': {
    backgroundColor: 'var(--bg-tertiary)',
    color: 'var(--text-muted)',
    border: 'none',
    borderRight: '1px solid var(--border-subtle)',
  },
  '.cm-activeLineGutter': {
    backgroundColor: 'rgba(245, 166, 35, 0.08)',
    color: 'var(--accent-primary)',
  },
  '.cm-lineNumbers .cm-gutterElement': {
    padding: '0 12px 0 8px',
    minWidth: '40px',
    fontFamily: 'var(--font-mono)',
    fontSize: '11px',
  },
}, { dark: true })

// Highlight style for markdown
const forgeHighlightStyle = HighlightStyle.define([
  { tag: tags.heading, color: 'var(--accent-primary)', fontWeight: '600' },
  { tag: tags.strong, color: '#e5c07b', fontWeight: '600' },
  { tag: tags.emphasis, color: '#c678dd', fontStyle: 'italic' },
  { tag: tags.link, color: '#61afef' },
  { tag: tags.url, color: '#61afef', textDecoration: 'underline' },
  { tag: tags.string, color: '#98c379' },
  { tag: tags.comment, color: 'var(--text-muted)' },
])

// Plugin for {{variable}} highlighting
const variableHighlighter = ViewPlugin.fromClass(class {
  decorations: DecorationSet
  constructor(view: EditorView) {
    this.decorations = this.buildDecorations(view)
  }
  update(update: ViewUpdate) {
    if (update.docChanged || update.viewportChanged) {
      this.decorations = this.buildDecorations(update.view)
    }
  }
  buildDecorations(view: EditorView): DecorationSet {
    const decorations: { from: number; to: number; decoration: Decoration }[] = []
    const doc = view.state.doc.toString()
    const regex = /\{\{\s*\w+\s*\}\}/g
    let match
    while ((match = regex.exec(doc)) !== null) {
      decorations.push({
        from: match.index,
        to: match.index + match[0].length,
        decoration: Decoration.mark({ class: styles.cmVariable }),
      })
    }
    return Decoration.set(decorations.map(d => d.decoration.range(d.from, d.to)))
  }
}, {
  decorations: v => v.decorations,
})

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
  const editorRef = useRef<HTMLDivElement>(null)
  const viewRef = useRef<EditorView | null>(null)
  const readOnlyCompartment = useRef(new Compartment())

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

  const onContentChange = useCallback((value: string) => {
    setContent(value)
  }, [])

  // Initialize CodeMirror
  useEffect(() => {
    if (!editorRef.current || loading || (error && !prompt)) return

    const updateListener = EditorView.updateListener.of((update: ViewUpdate) => {
      if (update.docChanged) {
        onContentChange(update.state.doc.toString())
      }
    })

    const state = EditorState.create({
      doc: content,
      extensions: [
        forgeTheme,
        syntaxHighlighting(forgeHighlightStyle),
        markdown(),
        variableHighlighter,
        EditorView.lineWrapping,
        updateListener,
        keymap.of([]),
        readOnlyCompartment.current.of(EditorState.readOnly.of(false)),
      ],
    })

    const view = new EditorView({
      state,
      parent: editorRef.current,
    })

    viewRef.current = view

    return () => {
      view.destroy()
      viewRef.current = null
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [loading, error, prompt])

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
          <div ref={editorRef} className={styles.editorContainer} data-testid="codemirror-editor" />
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
