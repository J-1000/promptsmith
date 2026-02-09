import { useState, useEffect, useMemo, useCallback } from 'react'
import {
  listPrompts,
  getPromptVersions,
  getAvailableModels,
  runPlayground,
  Prompt,
  Version,
  ModelInfo,
  PlaygroundRunResponse,
} from '../api'
import styles from './PlaygroundPage.module.css'

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

export function PlaygroundPage() {
  // Source mode
  const [sourceMode, setSourceMode] = useState<'library' | 'adhoc'>('adhoc')

  // Library mode state
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [selectedPrompt, setSelectedPrompt] = useState('')
  const [versions, setVersions] = useState<Version[]>([])
  const [selectedVersion, setSelectedVersion] = useState('')
  const [libraryContent, setLibraryContent] = useState('')

  // Ad-hoc mode state
  const [adhocContent, setAdhocContent] = useState('')

  // Shared state
  const [models, setModels] = useState<ModelInfo[]>([])
  const [selectedModel, setSelectedModel] = useState('')
  const [temperature, setTemperature] = useState(1.0)
  const [maxTokens, setMaxTokens] = useState(1024)
  const [variables, setVariables] = useState<Record<string, string>>({})

  // Execution state
  const [running, setRunning] = useState(false)
  const [result, setResult] = useState<PlaygroundRunResponse | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [showRendered, setShowRendered] = useState(false)

  // Active content depends on mode
  const activeContent = sourceMode === 'library' ? libraryContent : adhocContent

  // Extract variables from active content
  const detectedVars = useMemo(() => extractVariables(activeContent), [activeContent])

  // Sync variable inputs when detected vars change
  useEffect(() => {
    setVariables((prev) => {
      const next: Record<string, string> = {}
      for (const v of detectedVars) {
        next[v] = prev[v] || ''
      }
      return next
    })
  }, [detectedVars])

  // Load prompts and models on mount
  useEffect(() => {
    listPrompts().then(setPrompts).catch(() => {})
    getAvailableModels()
      .then((m) => {
        setModels(m)
        if (m.length > 0) setSelectedModel(m[0].id)
      })
      .catch(() => {})
  }, [])

  // Load versions when prompt changes
  useEffect(() => {
    if (!selectedPrompt) {
      setVersions([])
      setSelectedVersion('')
      setLibraryContent('')
      return
    }
    getPromptVersions(selectedPrompt)
      .then((v) => {
        setVersions(v)
        if (v.length > 0) {
          setSelectedVersion(v[0].version)
          setLibraryContent(v[0].content)
        }
      })
      .catch(() => {})
  }, [selectedPrompt])

  // Update library content when version changes
  const handleVersionChange = useCallback(
    (ver: string) => {
      setSelectedVersion(ver)
      const found = versions.find((v) => v.version === ver)
      if (found) setLibraryContent(found.content)
    },
    [versions]
  )

  const handleRun = async () => {
    setRunning(true)
    setError(null)
    setResult(null)

    try {
      const req =
        sourceMode === 'library' && selectedPrompt
          ? {
              prompt_name: selectedPrompt,
              version: selectedVersion || undefined,
              model: selectedModel,
              variables: Object.keys(variables).length > 0 ? variables : undefined,
              max_tokens: maxTokens,
              temperature,
            }
          : {
              content: adhocContent,
              model: selectedModel,
              variables: Object.keys(variables).length > 0 ? variables : undefined,
              max_tokens: maxTokens,
              temperature,
            }

      const res = await runPlayground(req)
      setResult(res)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Run failed')
    } finally {
      setRunning(false)
    }
  }

  const canRun =
    selectedModel && (sourceMode === 'adhoc' ? adhocContent.trim() : selectedPrompt) && !running

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Playground</h1>
      </div>

      <div className={styles.panels}>
        {/* Left panel: Input */}
        <div className={styles.panel}>
          <div className={styles.panelHeader}>
            <span className={styles.panelTitle}>Input</span>
          </div>
          <div className={styles.panelBody}>
            {/* Source toggle */}
            <div className={styles.sourceToggle}>
              <button
                className={`${styles.sourceToggleBtn} ${sourceMode === 'library' ? styles.sourceToggleBtnActive : ''}`}
                onClick={() => setSourceMode('library')}
              >
                From Library
              </button>
              <button
                className={`${styles.sourceToggleBtn} ${sourceMode === 'adhoc' ? styles.sourceToggleBtnActive : ''}`}
                onClick={() => setSourceMode('adhoc')}
              >
                Ad-hoc
              </button>
            </div>

            {/* Prompt source */}
            {sourceMode === 'library' ? (
              <>
                <div className={styles.field}>
                  <label className={styles.label}>Prompt</label>
                  <select
                    className={styles.select}
                    value={selectedPrompt}
                    onChange={(e) => setSelectedPrompt(e.target.value)}
                  >
                    <option value="">Select a prompt...</option>
                    {prompts.map((p) => (
                      <option key={p.name} value={p.name}>
                        {p.name}
                      </option>
                    ))}
                  </select>
                </div>
                {versions.length > 0 && (
                  <div className={styles.field}>
                    <label className={styles.label}>Version</label>
                    <select
                      className={styles.select}
                      value={selectedVersion}
                      onChange={(e) => handleVersionChange(e.target.value)}
                    >
                      {versions.map((v) => (
                        <option key={v.id} value={v.version}>
                          v{v.version}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
              </>
            ) : (
              <div className={styles.field}>
                <label className={styles.label}>Prompt Content</label>
                <textarea
                  className={styles.textarea}
                  value={adhocContent}
                  onChange={(e) => setAdhocContent(e.target.value)}
                  placeholder="Enter your prompt here... Use {{variable}} for variables."
                  rows={6}
                />
              </div>
            )}

            {/* Variables */}
            <div className={styles.field}>
              <label className={styles.label}>Variables</label>
              {detectedVars.length === 0 ? (
                <span className={styles.noVarsHint}>
                  No variables detected. Use {'{{varName}}'} syntax.
                </span>
              ) : (
                <div className={styles.variablesGrid}>
                  {detectedVars.map((v) => (
                    <div key={v} className={styles.variableRow}>
                      <span className={styles.variableLabel}>{`{{${v}}}`}</span>
                      <input
                        className={styles.input}
                        type="text"
                        value={variables[v] || ''}
                        onChange={(e) =>
                          setVariables((prev) => ({ ...prev, [v]: e.target.value }))
                        }
                        placeholder={v}
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>

            {/* Model + params */}
            <div className={styles.paramsRow}>
              <div className={styles.field}>
                <label className={styles.label}>Model</label>
                <select
                  className={styles.select}
                  value={selectedModel}
                  onChange={(e) => setSelectedModel(e.target.value)}
                >
                  {models.length === 0 && <option value="">No models available</option>}
                  {models.map((m) => (
                    <option key={m.id} value={m.id}>
                      {m.id}
                    </option>
                  ))}
                </select>
              </div>

              <div className={styles.field}>
                <div className={styles.sliderContainer}>
                  <div className={styles.sliderHeader}>
                    <label className={styles.label}>Temperature</label>
                    <span className={styles.sliderValue}>{temperature.toFixed(1)}</span>
                  </div>
                  <input
                    type="range"
                    className={styles.slider}
                    min="0"
                    max="2"
                    step="0.1"
                    value={temperature}
                    onChange={(e) => setTemperature(parseFloat(e.target.value))}
                  />
                </div>
              </div>

              <div className={styles.field}>
                <label className={styles.label}>Max Tokens</label>
                <input
                  className={styles.input}
                  type="number"
                  min={1}
                  max={16384}
                  value={maxTokens}
                  onChange={(e) => setMaxTokens(parseInt(e.target.value) || 1024)}
                />
              </div>
            </div>

            <button className={styles.runButton} onClick={handleRun} disabled={!canRun}>
              {running ? 'Running...' : 'Run'}
            </button>

            {error && <div className={styles.error}>{error}</div>}
          </div>
        </div>

        {/* Right panel: Output */}
        <div className={styles.panel}>
          <div className={styles.panelHeader}>
            <span className={styles.panelTitle}>Output</span>
            {result && (
              <button
                className={styles.renderedToggle}
                onClick={() => setShowRendered(!showRendered)}
              >
                {showRendered ? 'Hide' : 'Show'} rendered prompt
              </button>
            )}
          </div>
          <div className={styles.panelBody}>
            {showRendered && result && (
              <div className={styles.renderedPrompt}>{result.rendered_prompt}</div>
            )}

            {running ? (
              <div className={styles.spinner}>
                <div className={styles.spinnerDot} />
                <span className={styles.spinnerText}>Running completion...</span>
              </div>
            ) : result ? (
              <div className={styles.outputArea}>{result.output}</div>
            ) : (
              <div className={styles.emptyState}>
                <span className={styles.emptyIcon}>&#9655;</span>
                <span className={styles.emptyText}>Run a prompt to see output</span>
              </div>
            )}
          </div>

          {result && !running && (
            <div className={styles.statsBar}>
              <div className={styles.stat}>
                <span className={styles.statLabel}>Latency</span>
                <span className={styles.statValue}>{result.latency_ms}ms</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>In</span>
                <span className={styles.statValue}>{result.prompt_tokens}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>Out</span>
                <span className={styles.statValue}>{result.output_tokens}</span>
              </div>
              <div className={styles.stat}>
                <span className={styles.statLabel}>Cost</span>
                <span className={styles.statValue}>${result.cost.toFixed(4)}</span>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
