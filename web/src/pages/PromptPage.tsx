import { useParams, Link } from 'react-router-dom'
import { useState } from 'react'
import { DiffViewer } from '../components/DiffViewer'
import { TestResults, SuiteResult } from '../components/TestResults'
import styles from './PromptPage.module.css'

// Mock data - will be replaced with CLI/API integration
const mockVersions = [
  {
    version: '1.0.2',
    message: 'Add tone parameter for flexibility',
    date: '2024-01-15 14:32',
    tags: ['prod'],
  },
  {
    version: '1.0.1',
    message: 'Fix greeting for edge cases',
    date: '2024-01-12 09:15',
    tags: [],
  },
  {
    version: '1.0.0',
    message: 'Initial version of greeting prompt',
    date: '2024-01-10 16:45',
    tags: ['staging'],
  },
]

const mockContent = `---
name: greeting
description: A friendly greeting prompt
model_hint: gpt-4o
variables:
  - name: user_name
    type: string
    required: true
  - name: tone
    type: enum
    options: [formal, casual, friendly]
    default: friendly
---

You are a helpful assistant. Greet the user {{user_name}} in a {{tone}} manner.

Be warm and welcoming. Keep the greeting concise but personable.`

const mockDiff = `@@ -10,6 +10,9 @@
   - name: user_name
     type: string
     required: true
+  - name: tone
+    type: enum
+    options: [formal, casual, friendly]
+    default: friendly
 ---

-You are a helpful assistant. Greet the user {{user_name}}.
+You are a helpful assistant. Greet the user {{user_name}} in a {{tone}} manner.

-Be warm and welcoming.
+Be warm and welcoming. Keep the greeting concise but personable.`

const mockTestResults: SuiteResult = {
  suiteName: 'greeting-tests',
  promptName: 'greeting',
  version: '1.0.2',
  passed: 3,
  failed: 1,
  skipped: 1,
  total: 5,
  durationMs: 45,
  results: [
    {
      testName: 'basic-greeting',
      passed: true,
      skipped: false,
      durationMs: 8,
    },
    {
      testName: 'formal-tone',
      passed: true,
      skipped: false,
      durationMs: 12,
    },
    {
      testName: 'casual-tone',
      passed: true,
      skipped: false,
      durationMs: 10,
    },
    {
      testName: 'max-length-check',
      passed: false,
      skipped: false,
      durationMs: 11,
      failures: [
        {
          type: 'max_length',
          message: 'expected at most 50 characters, got 78',
          expected: '50',
          actual: '78',
        },
      ],
    },
    {
      testName: 'edge-case-empty-name',
      passed: false,
      skipped: true,
      durationMs: 0,
    },
  ],
}

export function PromptPage() {
  const { name } = useParams<{ name: string }>()
  const [selectedVersions, setSelectedVersions] = useState<string[]>([])
  const [view, setView] = useState<'content' | 'history' | 'diff' | 'tests'>('content')
  const [testResults, setTestResults] = useState<SuiteResult | null>(mockTestResults)
  const [isRunningTests, setIsRunningTests] = useState(false)

  const handleRunTests = () => {
    setIsRunningTests(true)
    // Simulate running tests
    setTimeout(() => {
      setTestResults(mockTestResults)
      setIsRunningTests(false)
    }, 1000)
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
          <span className={styles.version}>v{mockVersions[0].version}</span>
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
        </div>
      </div>

      <div className={styles.content}>
        {view === 'content' && (
          <div className={styles.codeBlock}>
            <div className={styles.codeHeader}>
              <span className={styles.fileName}>{name}.prompt</span>
            </div>
            <pre className={styles.code}>{mockContent}</pre>
          </div>
        )}

        {view === 'history' && (
          <div className={styles.history}>
            <p className={styles.historyHint}>
              Select two versions to compare
            </p>
            {mockVersions.map((v) => (
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
                      <span key={tag} className={styles.tag}>{tag}</span>
                    ))}
                  </div>
                  <p className={styles.versionMessage}>{v.message}</p>
                  <span className={styles.versionDate}>{v.date}</span>
                </div>
              </div>
            ))}
          </div>
        )}

        {view === 'diff' && selectedVersions.length === 2 && (
          <DiffViewer
            oldVersion={selectedVersions[0]}
            newVersion={selectedVersions[1]}
            diff={mockDiff}
          />
        )}

        {view === 'tests' && (
          <TestResults
            results={testResults}
            onRunTests={handleRunTests}
            isRunning={isRunningTests}
          />
        )}
      </div>
    </div>
  )
}
