import { useParams, Link, useNavigate } from 'react-router-dom'
import { useState, useEffect } from 'react'
import { createTwoFilesPatch } from 'diff'
import { DiffViewer } from '../components/DiffViewer'
import { TestResults, SuiteResult } from '../components/TestResults'
import { BenchmarkResults, BenchmarkResult } from '../components/BenchmarkResults'
import { GeneratePanel, GenerateResult, GenerationType } from '../components/GeneratePanel'
import { ConfirmDialog } from '../components/ConfirmDialog'
import { Toast } from '../components/Toast'
import {
  getPrompt,
  getPromptVersions,
  getPromptDiff,
  deletePrompt,
  createTag,
  deleteTag,
  runTest,
  runBenchmark,
  generateVariations,
  listTests,
  listBenchmarks,
  Prompt,
  Version,
  TestSuite,
  BenchmarkSuite,
} from '../api'
import styles from './PromptPage.module.css'

interface VersionDisplay {
  id: string
  version: string
  message: string
  date: string
  tags: string[]
  content: string
}

export function PromptPage() {
  const { name } = useParams<{ name: string }>()
  const navigate = useNavigate()
  const [prompt, setPrompt] = useState<Prompt | null>(null)
  const [versions, setVersions] = useState<VersionDisplay[]>([])
  const [currentContent, setCurrentContent] = useState<string>('')
  const [viewingVersion, setViewingVersion] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedVersions, setSelectedVersions] = useState<string[]>([])
  const [diffContent, setDiffContent] = useState<string>('')
  const [view, setView] = useState<'content' | 'history' | 'diff' | 'tests' | 'benchmarks' | 'generate'>('content')
  const [testResults, setTestResults] = useState<SuiteResult | null>(null)
  const [isRunningTests, setIsRunningTests] = useState(false)
  const [benchmarkResults, setBenchmarkResults] = useState<BenchmarkResult | null>(null)
  const [isRunningBenchmark, setIsRunningBenchmark] = useState(false)
  const [generateResults, setGenerateResults] = useState<GenerateResult | null>(null)
  const [isGenerating, setIsGenerating] = useState(false)
  const [copied, setCopied] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)
  const [toast, setToast] = useState<{ message: string; type: 'success' | 'error' | 'info' } | null>(null)
  const [newTagVersion, setNewTagVersion] = useState<string | null>(null)
  const [newTagName, setNewTagName] = useState('')
  const [affectedTests, setAffectedTests] = useState<TestSuite[]>([])
  const [affectedBenchmarks, setAffectedBenchmarks] = useState<BenchmarkSuite[]>([])

  const copyToClipboard = async () => {
    if (!currentContent) return
    try {
      await navigator.clipboard.writeText(currentContent)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch (err) {
      console.error('Failed to copy:', err)
    }
  }

  const handleDelete = async () => {
    if (!name) return
    try {
      await deletePrompt(name)
      navigate('/')
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to delete'
      setToast({ message: msg, type: 'error' })
      setShowDeleteConfirm(false)
    }
  }

  const refreshVersions = async () => {
    if (!name) return
    const versionsData = await getPromptVersions(name)
    const versionDisplays = versionsData.map((v: Version) => ({
      version: v.version,
      message: v.commit_message,
      date: new Date(v.created_at).toLocaleString(),
      tags: v.tags || [],
      content: v.content,
      id: v.id,
    }))
    setVersions(versionDisplays)
  }

  const handleAddTag = async (versionId: string) => {
    if (!name || !newTagName.trim()) return
    try {
      await createTag(name, newTagName.trim(), versionId)
      setNewTagVersion(null)
      setNewTagName('')
      await refreshVersions()
      setToast({ message: `Tag "${newTagName.trim()}" added`, type: 'success' })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to add tag'
      setToast({ message: msg, type: 'error' })
    }
  }

  const handleDeleteTag = async (tagName: string) => {
    if (!name) return
    try {
      await deleteTag(name, tagName)
      await refreshVersions()
      setToast({ message: `Tag "${tagName}" removed`, type: 'success' })
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : 'Failed to delete tag'
      setToast({ message: msg, type: 'error' })
    }
  }

  const handleVersionChange = (version: string) => {
    const versionData = versions.find((v) => v.version === version)
    if (versionData) {
      setViewingVersion(version)
      setCurrentContent(versionData.content)
    }
  }

  useEffect(() => {
    if (!name) return

    Promise.all([getPrompt(name), getPromptVersions(name)])
      .then(([promptData, versionsData]) => {
        setPrompt(promptData)
        const versionDisplays = versionsData.map((v: Version) => ({
          id: v.id,
          version: v.version,
          message: v.commit_message,
          date: new Date(v.created_at).toLocaleString(),
          tags: v.tags || [],
          content: v.content,
        }))
        setVersions(versionDisplays)
        if (versionDisplays.length > 0) {
          setCurrentContent(versionDisplays[0].content)
          setViewingVersion(versionDisplays[0].version)
        }
      })
      .catch((err) => setError(err.message))
      .finally(() => setLoading(false))
  }, [name])

  useEffect(() => {
    if (selectedVersions.length !== 2 || !name) return

    getPromptDiff(name, selectedVersions[0], selectedVersions[1])
      .then((data) => {
        const diff = createTwoFilesPatch(
          `v${data.v1.version}`,
          `v${data.v2.version}`,
          data.v1.content,
          data.v2.content
        )
        // Remove the file header lines
        const lines = diff.split('\n').slice(2)
        setDiffContent(lines.join('\n'))
      })
      .catch(console.error)
  }, [selectedVersions, name])

  useEffect(() => {
    if (view !== 'diff' || !name) return
    listTests().then((suites) => setAffectedTests(suites.filter(s => s.prompt === name))).catch(() => {})
    listBenchmarks().then((suites) => setAffectedBenchmarks(suites.filter(s => s.prompt === name))).catch(() => {})
  }, [view, name])

  const handleRunTests = async () => {
    if (!name) return
    setIsRunningTests(true)
    try {
      const result = await runTest(name)
      setTestResults({
        suiteName: result.suite_name,
        promptName: result.prompt_name,
        version: result.version,
        passed: result.passed,
        failed: result.failed,
        skipped: result.skipped,
        total: result.total,
        durationMs: result.duration_ms,
        results: result.results.map((r) => ({
          testName: r.test_name,
          passed: r.passed,
          skipped: r.skipped,
          output: r.output,
          failures: r.failures?.map((f) => ({
            type: f.type,
            message: f.message,
            expected: f.expected,
            actual: f.actual,
          })),
          error: r.error,
          durationMs: r.duration_ms,
        })),
      })
    } catch (err) {
      console.error('Failed to run tests:', err)
    } finally {
      setIsRunningTests(false)
    }
  }

  const handleRunBenchmark = async () => {
    if (!name) return
    setIsRunningBenchmark(true)
    try {
      const result = await runBenchmark(name)
      setBenchmarkResults({
        suiteName: result.suite_name,
        promptName: result.prompt_name,
        version: result.version,
        models: result.models.map((m) => ({
          model: m.model,
          runs: m.runs,
          latencyP50Ms: m.latency_p50_ms,
          latencyP99Ms: m.latency_p99_ms,
          latencyAvgMs: m.latency_p50_ms,
          totalTokensAvg: m.total_tokens_avg,
          promptTokens: 0,
          outputTokensAvg: m.total_tokens_avg,
          costPerRequest: m.cost_per_request,
          totalCost: m.cost_per_request * m.runs,
          errors: m.errors,
          errorRate: m.error_rate,
        })),
        durationMs: result.duration_ms,
      })
    } catch (err) {
      console.error('Failed to run benchmark:', err)
    } finally {
      setIsRunningBenchmark(false)
    }
  }

  const handleGenerate = async (type: GenerationType, count: number, goal: string) => {
    if (!currentContent) return
    setIsGenerating(true)
    try {
      const result = await generateVariations({
        type,
        prompt: currentContent,
        count,
        goal: goal || undefined,
      })
      setGenerateResults({
        original: result.original,
        variations: result.variations.map((v) => ({
          content: v.content,
          description: v.description,
          tokenDelta: v.token_delta,
        })),
        model: result.model,
        type: result.type as GenerationType,
        goal: result.goal,
      })
    } catch (err) {
      console.error('Failed to generate:', err)
    } finally {
      setIsGenerating(false)
    }
  }

  const toggleVersion = (version: string) => {
    setSelectedVersions((prev) => {
      if (prev.includes(version)) {
        return prev.filter((v) => v !== version)
      }
      if (prev.length >= 2) {
        return [prev[1], version]
      }
      return [...prev, version]
    })
  }

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading prompt...</div>
      </div>
    )
  }

  if (error) {
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
        <span className={styles.breadcrumbCurrent}>{name}</span>
      </div>

      <div className={styles.header}>
        <div className={styles.headerLeft}>
          <h1 className={styles.title}>{name}</h1>
          {prompt?.version && <span className={styles.version}>v{prompt.version}</span>}
          {versions.length > 0 && versions[0].tags.map((tag) => (
            <span key={tag} className={styles.headerTag}>{tag}</span>
          ))}
          <Link to={`/prompt/${name}/edit`} className={styles.editButton}>Edit</Link>
          <button className={styles.deleteButton} onClick={() => setShowDeleteConfirm(true)}>Delete</button>
        </div>
        <div className={styles.tabs}>
          <button
            className={`${styles.tab} ${view === 'content' ? styles.tabActive : ''}`}
            onClick={() => setView('content')}
          >
            Content
          </button>
          <button
            className={`${styles.tab} ${view === 'history' ? styles.tabActive : ''}`}
            onClick={() => setView('history')}
          >
            History
          </button>
          <button
            className={`${styles.tab} ${view === 'diff' ? styles.tabActive : ''}`}
            onClick={() => setView('diff')}
            disabled={selectedVersions.length < 2}
          >
            Diff {selectedVersions.length === 2 && `(${selectedVersions[0]} vs ${selectedVersions[1]})`}
          </button>
          <button
            className={`${styles.tab} ${view === 'tests' ? styles.tabActive : ''}`}
            onClick={() => setView('tests')}
          >
            Tests {testResults && (
              <span className={testResults.failed > 0 ? styles.testsFailed : styles.testsPassed}>
                {testResults.passed}/{testResults.total}
              </span>
            )}
          </button>
          <button
            className={`${styles.tab} ${view === 'benchmarks' ? styles.tabActive : ''}`}
            onClick={() => setView('benchmarks')}
          >
            Benchmarks {benchmarkResults && (
              <span className={styles.benchmarkCount}>
                {benchmarkResults.models.length}
              </span>
            )}
          </button>
          <button
            className={`${styles.tab} ${view === 'generate' ? styles.tabActive : ''}`}
            onClick={() => setView('generate')}
          >
            Generate
          </button>
        </div>
      </div>

      <div className={styles.content}>
        {view === 'content' && (
          <div className={styles.codeBlock}>
            <div className={styles.codeHeader}>
              <div className={styles.codeHeaderLeft}>
                <span className={styles.fileName}>{prompt?.file_path || `${name}.prompt`}</span>
                {versions.length > 1 && (
                  <select
                    className={styles.versionSelect}
                    value={viewingVersion}
                    onChange={(e) => handleVersionChange(e.target.value)}
                  >
                    {versions.map((v) => (
                      <option key={v.version} value={v.version}>
                        v{v.version}
                      </option>
                    ))}
                  </select>
                )}
              </div>
              <button className={styles.copyButton} onClick={copyToClipboard}>
                {copied ? 'Copied!' : 'Copy'}
              </button>
            </div>
            <pre className={styles.code}>{currentContent || 'No content'}</pre>
          </div>
        )}

        {view === 'history' && (
          <div className={styles.history}>
            {versions.length === 0 ? (
              <div className={styles.emptyHistory}>
                <p>No versions yet.</p>
                <p className={styles.hint}>Commit changes with: <code>promptsmith commit -m "message"</code></p>
              </div>
            ) : (
              <>
                <p className={styles.historyHint}>
                  Select two versions to compare
                </p>
                {versions.map((v) => (
                  <div
                    key={v.version}
                    className={`${styles.versionRow} ${selectedVersions.includes(v.version) ? styles.versionSelected : ''}`}
                    onClick={() => toggleVersion(v.version)}
                  >
                    <div className={styles.versionCheck}>
                      {selectedVersions.includes(v.version) && (
                        <span className={styles.checkMark}>&#10003;</span>
                      )}
                    </div>
                    <div className={styles.versionInfo}>
                      <div className={styles.versionHeader}>
                        <span className={styles.versionNumber}>v{v.version}</span>
                        {v.tags.map((tag) => (
                          <span key={tag} className={styles.tag}>
                            {tag}
                            <button
                              className={styles.tagDelete}
                              onClick={(e) => { e.stopPropagation(); handleDeleteTag(tag) }}
                              aria-label={`Remove tag ${tag}`}
                            >
                              &times;
                            </button>
                          </span>
                        ))}
                        {newTagVersion === v.id ? (
                          <span className={styles.tagInput} onClick={(e) => e.stopPropagation()}>
                            <input
                              type="text"
                              className={styles.tagInputField}
                              value={newTagName}
                              onChange={(e) => setNewTagName(e.target.value)}
                              onKeyDown={(e) => {
                                if (e.key === 'Enter') handleAddTag(v.id)
                                if (e.key === 'Escape') { setNewTagVersion(null); setNewTagName('') }
                              }}
                              placeholder="tag name"
                              autoFocus
                            />
                          </span>
                        ) : (
                          <button
                            className={styles.addTagButton}
                            onClick={(e) => { e.stopPropagation(); setNewTagVersion(v.id); setNewTagName('') }}
                          >
                            + tag
                          </button>
                        )}
                      </div>
                      <p className={styles.versionMessage}>{v.message}</p>
                      <span className={styles.versionDate}>{v.date}</span>
                    </div>
                  </div>
                ))}
              </>
            )}
          </div>
        )}

        {view === 'diff' && selectedVersions.length === 2 && (
          <>
            <DiffViewer
              oldVersion={selectedVersions[0]}
              newVersion={selectedVersions[1]}
              diff={diffContent}
            />
            {(affectedTests.length > 0 || affectedBenchmarks.length > 0) && (
              <div className={styles.impactSection}>
                <h3 className={styles.impactTitle}>Change Impact</h3>
                <p className={styles.impactHint}>These items use this prompt and may be affected by changes:</p>
                <div className={styles.impactList}>
                  {affectedTests.map(t => (
                    <Link key={t.name} to={`/tests/${t.name}`} className={styles.impactItem}>
                      <span className={styles.impactIcon}>T</span>
                      <span>{t.name}</span>
                      <span className={styles.impactCount}>{t.test_count} tests</span>
                    </Link>
                  ))}
                  {affectedBenchmarks.map(b => (
                    <Link key={b.name} to={`/benchmarks/${b.name}`} className={styles.impactItem}>
                      <span className={styles.impactIcon}>B</span>
                      <span>{b.name}</span>
                    </Link>
                  ))}
                </div>
              </div>
            )}
          </>
        )}

        {view === 'tests' && (
          <TestResults
            results={testResults}
            onRunTests={handleRunTests}
            isRunning={isRunningTests}
          />
        )}

        {view === 'benchmarks' && (
          <BenchmarkResults
            results={benchmarkResults}
            onRunBenchmark={handleRunBenchmark}
            isRunning={isRunningBenchmark}
          />
        )}

        {view === 'generate' && (
          <GeneratePanel
            results={generateResults}
            onGenerate={handleGenerate}
            isGenerating={isGenerating}
          />
        )}
      </div>

      {showDeleteConfirm && (
        <ConfirmDialog
          title="Delete Prompt"
          message={`Are you sure you want to delete "${name}"? This cannot be undone.`}
          confirmLabel="Delete"
          variant="danger"
          onConfirm={handleDelete}
          onCancel={() => setShowDeleteConfirm(false)}
        />
      )}

      {toast && (
        <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />
      )}
    </div>
  )
}
