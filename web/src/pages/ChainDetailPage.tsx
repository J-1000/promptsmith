import { useState, useEffect, useMemo } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import {
  getChain,
  deleteChain,
  saveChainSteps,
  runChain,
  listPrompts,
  getAvailableModels,
  ChainDetail,
  ChainStepInput,
  ChainStepRunResult,
  Prompt,
  ModelInfo,
} from '../api'
import styles from './ChainDetailPage.module.css'

interface StepDraft {
  prompt_name: string
  output_key: string
  mappings: { key: string; value: string }[]
}

export function ChainDetailPage() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const [chain, setChain] = useState<ChainDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [prompts, setPrompts] = useState<Prompt[]>([])
  const [models, setModels] = useState<ModelInfo[]>([])

  // Steps editor
  const [steps, setSteps] = useState<StepDraft[]>([])
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)

  // Run panel
  const [runInputs, setRunInputs] = useState<Record<string, string>>({})
  const [selectedModel, setSelectedModel] = useState('gpt-4o-mini')
  const [running, setRunning] = useState(false)
  const [runError, setRunError] = useState<string | null>(null)
  const [stepResults, setStepResults] = useState<ChainStepRunResult[]>([])
  const [finalOutput, setFinalOutput] = useState<string | null>(null)
  const [expandedSteps, setExpandedSteps] = useState<Set<number>>(new Set())

  useEffect(() => {
    if (!name) return
    Promise.all([
      getChain(name),
      listPrompts(),
      getAvailableModels().catch(() => []),
    ])
      .then(([chainData, promptList, modelList]) => {
        setChain(chainData)
        setPrompts(promptList)
        setModels(modelList)
        setSteps(
          chainData.steps.map((s) => ({
            prompt_name: s.prompt_name,
            output_key: s.output_key,
            mappings: Object.entries(s.input_mapping || {}).map(([k, v]) => ({
              key: k,
              value: v,
            })),
          }))
        )
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [name])

  // Auto-detect required inputs from mappings
  const requiredInputs = useMemo(() => {
    const inputs = new Set<string>()
    for (const step of steps) {
      for (const m of step.mappings) {
        const match = m.value.match(/^\{\{input\.(\w+)\}\}$/)
        if (match) inputs.add(match[1])
      }
    }
    return Array.from(inputs)
  }, [steps])

  const handleAddStep = () => {
    setSteps([
      ...steps,
      { prompt_name: '', output_key: '', mappings: [{ key: '', value: '' }] },
    ])
    setDirty(true)
  }

  const handleRemoveStep = (idx: number) => {
    setSteps(steps.filter((_, i) => i !== idx))
    setDirty(true)
  }

  const handleMoveStep = (idx: number, dir: -1 | 1) => {
    const newIdx = idx + dir
    if (newIdx < 0 || newIdx >= steps.length) return
    const newSteps = [...steps]
    ;[newSteps[idx], newSteps[newIdx]] = [newSteps[newIdx], newSteps[idx]]
    setSteps(newSteps)
    setDirty(true)
  }

  const updateStep = (idx: number, field: keyof StepDraft, value: any) => {
    const newSteps = [...steps]
    newSteps[idx] = { ...newSteps[idx], [field]: value }
    setSteps(newSteps)
    setDirty(true)
  }

  const addMapping = (stepIdx: number) => {
    const newSteps = [...steps]
    newSteps[stepIdx] = {
      ...newSteps[stepIdx],
      mappings: [...newSteps[stepIdx].mappings, { key: '', value: '' }],
    }
    setSteps(newSteps)
    setDirty(true)
  }

  const updateMapping = (
    stepIdx: number,
    mapIdx: number,
    field: 'key' | 'value',
    val: string
  ) => {
    const newSteps = [...steps]
    const newMappings = [...newSteps[stepIdx].mappings]
    newMappings[mapIdx] = { ...newMappings[mapIdx], [field]: val }
    newSteps[stepIdx] = { ...newSteps[stepIdx], mappings: newMappings }
    setSteps(newSteps)
    setDirty(true)
  }

  const removeMapping = (stepIdx: number, mapIdx: number) => {
    const newSteps = [...steps]
    newSteps[stepIdx] = {
      ...newSteps[stepIdx],
      mappings: newSteps[stepIdx].mappings.filter((_, i) => i !== mapIdx),
    }
    setSteps(newSteps)
    setDirty(true)
  }

  const handleSave = async () => {
    if (!name) return
    setSaving(true)
    try {
      const stepInputs: ChainStepInput[] = steps.map((s, i) => ({
        step_order: i + 1,
        prompt_name: s.prompt_name,
        output_key: s.output_key,
        input_mapping: Object.fromEntries(
          s.mappings.filter((m) => m.key).map((m) => [m.key, m.value])
        ),
      }))
      await saveChainSteps(name, stepInputs)
      setDirty(false)
      // Reload chain
      const updated = await getChain(name)
      setChain(updated)
    } catch (err: any) {
      setError(err.message)
    } finally {
      setSaving(false)
    }
  }

  const handleRun = async () => {
    if (!name) return
    setRunning(true)
    setRunError(null)
    setStepResults([])
    setFinalOutput(null)
    try {
      const result = await runChain(name, runInputs, selectedModel)
      setStepResults(result.results)
      setFinalOutput(result.final_output)
    } catch (err: any) {
      setRunError(err.message)
    } finally {
      setRunning(false)
    }
  }

  const handleDelete = async () => {
    if (!name || !confirm(`Delete chain "${name}"? This cannot be undone.`)) return
    try {
      await deleteChain(name)
      navigate('/chains')
    } catch (err: any) {
      setError(err.message)
    }
  }

  const toggleExpanded = (stepOrder: number) => {
    setExpandedSteps((prev) => {
      const next = new Set(prev)
      if (next.has(stepOrder)) next.delete(stepOrder)
      else next.add(stepOrder)
      return next
    })
  }

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading chain...</div>
      </div>
    )
  }

  if (error && !chain) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>
          <p>Failed to load chain: {error}</p>
        </div>
      </div>
    )
  }

  if (!chain) return null

  return (
    <div className={styles.container}>
      <div className={styles.breadcrumb}>
        <Link to="/chains">Chains</Link>
        <span>/</span>
        <span>{chain.name}</span>
      </div>

      <div className={styles.headerRow}>
        <div className={styles.headerInfo}>
          <h1 className={styles.title}>{chain.name}</h1>
          {chain.description && (
            <p className={styles.description}>{chain.description}</p>
          )}
        </div>
        <button className={styles.deleteButton} onClick={handleDelete}>
          Delete
        </button>
      </div>

      {/* Steps Editor */}
      <div className={styles.section}>
        <div className={styles.sectionHeader}>
          <h2 className={styles.sectionTitle}>Steps</h2>
          <button className={styles.addStepButton} onClick={handleAddStep}>
            + Add Step
          </button>
        </div>

        {steps.length === 0 ? (
          <div className={styles.emptySteps}>
            No steps configured. Add a step to start building your chain.
          </div>
        ) : (
          <div className={styles.stepsList}>
            {steps.map((step, idx) => (
              <div key={idx}>
                {idx > 0 && (
                  <div className={styles.connector}>
                    <div className={styles.connectorLine} />
                  </div>
                )}
                <div className={styles.stepCard}>
                  <div className={styles.stepHeader}>
                    <span className={styles.stepNumber}>Step {idx + 1}</span>
                    <div className={styles.stepActions}>
                      <button
                        className={styles.stepActionBtn}
                        onClick={() => handleMoveStep(idx, -1)}
                        disabled={idx === 0}
                        title="Move up"
                      >
                        ↑
                      </button>
                      <button
                        className={styles.stepActionBtn}
                        onClick={() => handleMoveStep(idx, 1)}
                        disabled={idx === steps.length - 1}
                        title="Move down"
                      >
                        ↓
                      </button>
                      <button
                        className={styles.stepActionBtn}
                        onClick={() => handleRemoveStep(idx)}
                        title="Remove step"
                      >
                        ×
                      </button>
                    </div>
                  </div>

                  <div className={styles.stepFields}>
                    <div className={styles.stepField}>
                      <label className={styles.stepLabel}>Prompt</label>
                      <select
                        className={styles.stepSelect}
                        value={step.prompt_name}
                        onChange={(e) =>
                          updateStep(idx, 'prompt_name', e.target.value)
                        }
                      >
                        <option value="">Select prompt...</option>
                        {prompts.map((p) => (
                          <option key={p.id} value={p.name}>
                            {p.name}
                          </option>
                        ))}
                      </select>
                    </div>
                    <div className={styles.stepField}>
                      <label className={styles.stepLabel}>Output Key</label>
                      <input
                        className={styles.stepInput}
                        placeholder="e.g. summary"
                        value={step.output_key}
                        onChange={(e) =>
                          updateStep(idx, 'output_key', e.target.value)
                        }
                      />
                    </div>
                  </div>

                  <div className={styles.stepFieldFull}>
                    <label className={styles.stepLabel}>Input Mapping</label>
                    {step.mappings.map((m, mIdx) => (
                      <div key={mIdx} className={styles.mappingRow}>
                        <input
                          className={styles.mappingKey}
                          placeholder="variable"
                          value={m.key}
                          onChange={(e) =>
                            updateMapping(idx, mIdx, 'key', e.target.value)
                          }
                        />
                        <span className={styles.mappingArrow}>←</span>
                        <input
                          className={styles.mappingValue}
                          placeholder='{{input.var}} or {{steps.key.output}}'
                          value={m.value}
                          onChange={(e) =>
                            updateMapping(idx, mIdx, 'value', e.target.value)
                          }
                        />
                        <button
                          className={styles.removeMappingBtn}
                          onClick={() => removeMapping(idx, mIdx)}
                        >
                          ×
                        </button>
                      </div>
                    ))}
                    <button
                      className={styles.addMappingBtn}
                      onClick={() => addMapping(idx)}
                    >
                      + mapping
                    </button>
                  </div>
                </div>
              </div>
            ))}
          </div>
        )}

        {dirty && (
          <div className={styles.saveBar}>
            <button
              className={styles.saveButton}
              onClick={handleSave}
              disabled={saving}
            >
              {saving ? 'Saving...' : 'Save Steps'}
            </button>
          </div>
        )}
      </div>

      {/* Run Panel */}
      <div className={styles.section}>
        <h2 className={styles.sectionTitle}>Run Chain</h2>
        <div className={styles.runPanel}>
          {requiredInputs.length > 0 && (
            <div className={styles.runInputs}>
              {requiredInputs.map((key) => (
                <div key={key} className={styles.runInputRow}>
                  <label className={styles.runInputLabel}>{key}</label>
                  <input
                    className={styles.runInputField}
                    placeholder={`Enter ${key}...`}
                    value={runInputs[key] || ''}
                    onChange={(e) =>
                      setRunInputs({ ...runInputs, [key]: e.target.value })
                    }
                  />
                </div>
              ))}
            </div>
          )}

          <div className={styles.runActions}>
            <select
              className={styles.modelSelect}
              value={selectedModel}
              onChange={(e) => setSelectedModel(e.target.value)}
            >
              {models.length > 0 ? (
                models.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.id}
                  </option>
                ))
              ) : (
                <>
                  <option value="gpt-4o-mini">gpt-4o-mini</option>
                  <option value="gpt-4o">gpt-4o</option>
                  <option value="claude-sonnet-4-5-20250929">claude-sonnet-4-5</option>
                </>
              )}
            </select>
            <button
              className={styles.runButton}
              onClick={handleRun}
              disabled={running || steps.length === 0}
            >
              {running ? 'Running...' : 'Run Chain'}
            </button>
          </div>

          {runError && (
            <div className={styles.runError}>{runError}</div>
          )}

          {stepResults.length > 0 && (
            <div className={styles.results}>
              {stepResults.map((result) => (
                <div key={result.step_order} className={styles.resultStep}>
                  <div
                    className={styles.resultStepHeader}
                    onClick={() => toggleExpanded(result.step_order)}
                  >
                    <span className={styles.resultStepTitle}>
                      Step {result.step_order}: {result.prompt_name} → {result.output_key}
                    </span>
                    <span className={styles.resultStepDuration}>
                      {result.duration_ms}ms
                    </span>
                  </div>
                  {expandedSteps.has(result.step_order) && (
                    <div className={styles.resultStepBody}>
                      <div>
                        <div className={styles.resultLabel}>Rendered Prompt</div>
                        <div className={styles.resultContent}>
                          {result.rendered_prompt}
                        </div>
                      </div>
                      <div>
                        <div className={styles.resultLabel}>Output</div>
                        <div className={styles.resultContent}>
                          {result.output}
                        </div>
                      </div>
                    </div>
                  )}
                </div>
              ))}

              {finalOutput && (
                <div className={styles.finalOutput}>
                  <div className={styles.finalOutputLabel}>Final Output</div>
                  <div className={styles.finalOutputContent}>{finalOutput}</div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
