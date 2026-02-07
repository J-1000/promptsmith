import { useState, useEffect } from 'react'
import { getProject, getSyncConfig, Project, SyncConfig } from '../api'
import styles from './SettingsPage.module.css'

interface ProviderConfig {
  name: string
  envVar: string
  status: 'configured' | 'not_configured'
}

const PROVIDERS: ProviderConfig[] = [
  { name: 'OpenAI', envVar: 'OPENAI_API_KEY', status: 'not_configured' },
  { name: 'Anthropic', envVar: 'ANTHROPIC_API_KEY', status: 'not_configured' },
  { name: 'Google', envVar: 'GOOGLE_API_KEY', status: 'not_configured' },
]

export function SettingsPage() {
  const [project, setProject] = useState<Project | null>(null)
  const [syncConfig, setSyncConfig] = useState<SyncConfig | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    Promise.all([
      getProject().then(setProject).catch((err) => setError(err.message)),
      getSyncConfig().then(setSyncConfig).catch(() => {}),
    ]).finally(() => setLoading(false))
  }, [])

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading settings...</div>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <h1 className={styles.title}>Settings</h1>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Project</h2>
        <div className={styles.card}>
          {error ? (
            <div className={styles.errorInline}>
              <p>Could not load project info: {error}</p>
              <p className={styles.hint}>Make sure the server is running: <code>promptsmith serve</code></p>
            </div>
          ) : (
            <div className={styles.fieldGroup}>
              <div className={styles.field}>
                <label className={styles.fieldLabel}>Name</label>
                <span className={styles.fieldValue}>{project?.name || '—'}</span>
              </div>
              <div className={styles.field}>
                <label className={styles.fieldLabel}>ID</label>
                <code className={styles.fieldCode}>{project?.id || '—'}</code>
              </div>
            </div>
          )}
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>LLM Providers</h2>
        <div className={styles.card}>
          <p className={styles.sectionHint}>
            API keys are configured via environment variables. Set them in your shell or <code>.env</code> file.
          </p>
          <div className={styles.providerList}>
            {PROVIDERS.map((provider) => (
              <div key={provider.name} className={styles.providerRow}>
                <div className={styles.providerInfo}>
                  <span className={styles.providerName}>{provider.name}</span>
                  <code className={styles.envVar}>{provider.envVar}</code>
                </div>
                <span className={styles.statusBadge} data-status={provider.status}>
                  {provider.status === 'configured' ? 'Configured' : 'Not set'}
                </span>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>Sync</h2>
        <div className={styles.card}>
          {syncConfig && syncConfig.status === 'configured' ? (
            <>
              <div className={styles.fieldGroup}>
                {syncConfig.team && (
                  <div className={styles.field}>
                    <label className={styles.fieldLabel}>Team</label>
                    <span className={styles.fieldValue}>{syncConfig.team}</span>
                  </div>
                )}
                {syncConfig.remote && (
                  <div className={styles.field}>
                    <label className={styles.fieldLabel}>Remote</label>
                    <code className={styles.fieldCode}>{syncConfig.remote}</code>
                  </div>
                )}
                <div className={styles.field}>
                  <label className={styles.fieldLabel}>Auto-push</label>
                  <span className={styles.statusBadge} data-status={syncConfig.auto_push ? 'configured' : 'not_configured'}>
                    {syncConfig.auto_push ? 'Enabled' : 'Disabled'}
                  </span>
                </div>
                <div className={styles.field}>
                  <label className={styles.fieldLabel}>Status</label>
                  <span className={styles.statusBadge} data-status="configured">Connected</span>
                </div>
              </div>
            </>
          ) : (
            <>
              <p className={styles.sectionHint}>
                Configure cloud sync with the CLI:
              </p>
              <div className={styles.cliCommands}>
                <div className={styles.cliCommand}>
                  <code>promptsmith config sync.remote &lt;url&gt;</code>
                  <span className={styles.cliDesc}>Set remote server URL</span>
                </div>
                <div className={styles.cliCommand}>
                  <code>promptsmith config sync.team &lt;team&gt;</code>
                  <span className={styles.cliDesc}>Set team for collaboration</span>
                </div>
                <div className={styles.cliCommand}>
                  <code>promptsmith config sync.auto_push true</code>
                  <span className={styles.cliDesc}>Auto-push on commit</span>
                </div>
                <div className={styles.cliCommand}>
                  <code>promptsmith login</code>
                  <span className={styles.cliDesc}>Authenticate with cloud</span>
                </div>
              </div>
            </>
          )}
        </div>
      </section>

      <section className={styles.section}>
        <h2 className={styles.sectionTitle}>About</h2>
        <div className={styles.card}>
          <div className={styles.about}>
            <span className={styles.aboutLogo}>&#9881;</span>
            <span className={styles.aboutName}>PromptSmith</span>
            <span className={styles.aboutVersion}>v0.1.0</span>
          </div>
          <p className={styles.aboutDesc}>
            The GitHub Copilot for Prompt Engineering. Version, test, and benchmark your LLM prompts.
          </p>
        </div>
      </section>
    </div>
  )
}
