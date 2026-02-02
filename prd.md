# PromptSmith — Product Requirements Document

**The GitHub Copilot for Prompt Engineering**

---

## Executive Summary

PromptSmith is a developer-focused tool that brings software engineering best practices to prompt engineering. It combines a web dashboard with a powerful CLI, enabling teams to version, test, iterate, and benchmark their LLM prompts with the same rigor they apply to code.

---

## Problem Statement

Prompt engineering today is chaotic:

- **No version control**: Prompts live in random text files, Notion docs, or buried in application code
- **No testing**: Changes are deployed blind with no regression testing
- **No benchmarking**: Teams guess which model performs best for their use case
- **No collaboration**: Prompt iterations happen in silos without shared context
- **No reproducibility**: "It worked yesterday" is a common refrain

Developers need tooling that treats prompts as first-class software artifacts.

---

## Target Users

### Primary: Application Developers
Engineers integrating LLMs into products who need to manage prompts across environments (dev/staging/prod) and track changes over time.

### Secondary: AI/ML Engineers
Teams fine-tuning and optimizing prompts for specific tasks who need systematic benchmarking and A/B testing capabilities.

### Tertiary: Technical Product Managers
Non-engineers who write prompts and need visibility into prompt performance without touching code.

---

## Core Features

### 1. Versioned Prompts

**Goal**: Git-like version control specifically designed for prompts.

| Capability | Description |
|------------|-------------|
| Semantic versioning | `prompt@1.2.3` addressing for all prompts |
| Branching | Create experimental branches without affecting production |
| Commit history | Full audit trail with commit messages |
| Rollback | One-click revert to any previous version |
| Tags | Mark versions as `prod`, `staging`, `experimental` |

**CLI Usage**:
```bash
promptsmith init
promptsmith add summarizer.prompt
promptsmith commit -m "Improved tone for customer-facing summary"
promptsmith tag v1.2.0 --env prod
```

**Web UI**: Visual commit graph, side-by-side version comparison, one-click deployments.

---

### 2. Prompt Diffing

**Goal**: Meaningful diffs that highlight semantic changes, not just text changes.

| Capability | Description |
|------------|-------------|
| Syntax-aware diffing | Highlights variables, instructions, and examples separately |
| Side-by-side view | Compare any two versions |
| Inline annotations | Add comments to specific diff sections |
| Change impact preview | Show which tests would be affected |

**CLI Usage**:
```bash
promptsmith diff v1.1.0 v1.2.0
promptsmith diff HEAD~3 HEAD --format=unified
```

**Web UI**: Rich diff viewer with collapsible sections, change highlighting, and annotation threads.

---

### 3. Auto-Test Prompts

**Goal**: CI/CD-style testing for prompt changes.

| Capability | Description |
|------------|-------------|
| Test suites | Define expected inputs/outputs for prompts |
| Assertion types | Exact match, contains, regex, semantic similarity, JSON schema |
| Snapshot testing | Detect unexpected output changes |
| Regression detection | Flag when changes break existing behavior |
| CI integration | GitHub Actions, GitLab CI, etc. |

**Test Definition** (`tests/summarizer.test.yaml`):
```yaml
prompt: summarizer@latest
tests:
  - name: "Short article summarization"
    input:
      article: "{{fixtures/short-article.txt}}"
    assertions:
      - type: max_length
        value: 280
      - type: contains
        value: "key finding"
      - type: semantic_similarity
        reference: "{{fixtures/expected-summary.txt}}"
        threshold: 0.85

  - name: "Handles empty input gracefully"
    input:
      article: ""
    assertions:
      - type: not_contains
        value: "error"
```

**CLI Usage**:
```bash
promptsmith test                    # Run all tests
promptsmith test summarizer         # Test specific prompt
promptsmith test --watch            # Re-run on changes
promptsmith test --update-snapshots # Update baseline
```

---

### 4. Generate Prompt Variations with LLMs

**Goal**: AI-assisted prompt improvement and exploration.

| Capability | Description |
|------------|-------------|
| Variation generation | Create N variations optimized for different goals |
| Style transfer | Rewrite prompts for different tones/audiences |
| Compression | Reduce token count while preserving behavior |
| Expansion | Add detail, examples, or edge case handling |
| Translation | Convert prompts between languages |

**CLI Usage**:
```bash
promptsmith generate variations summarizer.prompt --count=5 --goal="reduce tokens"
promptsmith generate compress summarizer.prompt --target-reduction=30%
promptsmith generate expand summarizer.prompt --add-examples=3
```

**Web UI**: Side-by-side comparison of generated variations with instant test results.

---

### 5. Benchmark Performance vs. Different Models

**Goal**: Data-driven model selection and cost optimization.

| Capability | Description |
|------------|-------------|
| Multi-model execution | Run prompts against GPT-4, Claude, Gemini, Llama, etc. |
| Metrics collection | Latency, token usage, cost, quality scores |
| Quality evaluation | LLM-as-judge, human rating interface, custom scorers |
| Cost analysis | $/1K requests, $/quality-point |
| Visualization | Charts comparing models across dimensions |

**Benchmark Definition** (`benchmarks/summarizer.bench.yaml`):
```yaml
prompt: summarizer@latest
models:
  - gpt-4o
  - gpt-4o-mini
  - claude-sonnet-4-20250514
  - claude-haiku
  - gemini-1.5-pro
  - llama-3.1-70b

dataset: fixtures/benchmark-articles.jsonl
runs_per_model: 50

metrics:
  - latency_p50
  - latency_p99
  - total_tokens
  - cost_per_request
  - quality_score:
      evaluator: llm-judge
      criteria: "accuracy, completeness, conciseness"
```

**CLI Usage**:
```bash
promptsmith benchmark summarizer --models=all
promptsmith benchmark summarizer --models=gpt-4o,claude-sonnet --runs=100
promptsmith benchmark compare report-jan.json report-feb.json
```

**Web UI**: Interactive dashboards with model comparison charts, cost projections, and recommendation engine.

---

## Technical Architecture

### System Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         PromptSmith                              │
├─────────────────────────────────────────────────────────────────┤
│                                                                  │
│   ┌─────────────┐         ┌─────────────────────────────────┐   │
│   │             │         │         Next.js Web App          │   │
│   │   CLI       │ ◄─────► │  ┌─────────────────────────────┐ │   │
│   │  (Go/TS)    │   API   │  │   Dashboard   │   Editor    │ │   │
│   │             │         │  │   Diff View   │   Tests     │ │   │
│   └─────────────┘         │  │   Benchmarks  │   Settings  │ │   │
│         │                 │  └─────────────────────────────┘ │   │
│         │                 └─────────────────────────────────────┘│
│         │                              │                         │
│         ▼                              ▼                         │
│   ┌─────────────────────────────────────────────────────────┐   │
│   │                      API Layer                           │   │
│   │            (Next.js API Routes / tRPC)                   │   │
│   └─────────────────────────────────────────────────────────┘   │
│                              │                                   │
│         ┌────────────────────┼────────────────────┐             │
│         ▼                    ▼                    ▼             │
│   ┌───────────┐      ┌─────────────┐      ┌─────────────┐       │
│   │  SQLite/  │      │    LLM      │      │   Queue     │       │
│   │  Turso    │      │  Providers  │      │  (Optional) │       │
│   └───────────┘      └─────────────┘      └─────────────┘       │
│                              │                                   │
│              ┌───────────────┼───────────────┐                  │
│              ▼               ▼               ▼                  │
│         ┌────────┐     ┌────────┐     ┌────────┐                │
│         │OpenAI  │     │Anthropic│    │ Google │   ...          │
│         └────────┘     └────────┘     └────────┘                │
│                                                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Technology Choices

| Component | Technology | Rationale |
|-----------|------------|-----------|
| **Web App** | Next.js 14+ (App Router) | Full-stack React, API routes, edge-ready |
| **CLI** | Go | Fast startup, single binary, excellent DX |
| **Database** | SQLite (local) / Turso (cloud) | Zero-config local, seamless cloud sync |
| **Auth** | NextAuth.js | GitHub/Google OAuth, simple setup |
| **LLM Integration** | Vercel AI SDK | Unified interface, streaming support |
| **Testing** | Vitest (web), Go testing (CLI) | Modern, fast, good DX |
| **Styling** | Tailwind CSS + shadcn/ui | Rapid development, consistent design |

### Data Model

```sql
-- Core entities
CREATE TABLE projects (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE prompts (
  id TEXT PRIMARY KEY,
  project_id TEXT REFERENCES projects(id),
  name TEXT NOT NULL,
  description TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE prompt_versions (
  id TEXT PRIMARY KEY,
  prompt_id TEXT REFERENCES prompts(id),
  version TEXT NOT NULL,          -- semver: 1.2.3
  content TEXT NOT NULL,
  variables JSON,                 -- extracted template variables
  metadata JSON,                  -- model hints, token estimates
  parent_version_id TEXT,         -- for branching
  commit_message TEXT,
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
  created_by TEXT
);

CREATE TABLE tags (
  id TEXT PRIMARY KEY,
  prompt_id TEXT REFERENCES prompts(id),
  version_id TEXT REFERENCES prompt_versions(id),
  name TEXT NOT NULL,             -- prod, staging, v1.0.0
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Testing
CREATE TABLE test_suites (
  id TEXT PRIMARY KEY,
  prompt_id TEXT REFERENCES prompts(id),
  name TEXT NOT NULL,
  config JSON NOT NULL            -- test definitions
);

CREATE TABLE test_runs (
  id TEXT PRIMARY KEY,
  suite_id TEXT REFERENCES test_suites(id),
  version_id TEXT REFERENCES prompt_versions(id),
  status TEXT,                    -- pending, running, passed, failed
  results JSON,
  started_at DATETIME,
  completed_at DATETIME
);

-- Benchmarking
CREATE TABLE benchmarks (
  id TEXT PRIMARY KEY,
  prompt_id TEXT REFERENCES prompts(id),
  config JSON NOT NULL            -- models, metrics, dataset
);

CREATE TABLE benchmark_runs (
  id TEXT PRIMARY KEY,
  benchmark_id TEXT REFERENCES benchmarks(id),
  version_id TEXT REFERENCES prompt_versions(id),
  results JSON,                   -- per-model metrics
  created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## CLI Specification

### Installation

```bash
# Homebrew (macOS/Linux)
brew install promptsmith/tap/promptsmith

# Go install
go install github.com/promptsmith/cli@latest

# npm (TypeScript alternative)
npm install -g @promptsmith/cli
```

### Command Reference

```
promptsmith <command> [options]

Commands:
  init              Initialize a new PromptSmith project
  add <file>        Track a new prompt file
  commit            Record changes to prompts
  diff              Show changes between versions
  log               Show commit history
  tag               Create, list, or delete tags
  checkout          Switch to a different version
  
  test              Run prompt tests
  generate          Generate prompt variations
  benchmark         Run model benchmarks
  
  push              Sync local changes to cloud
  pull              Fetch latest from cloud
  
  login             Authenticate with PromptSmith cloud
  config            Manage configuration

Options:
  -h, --help        Show help
  -v, --version     Show version
  --verbose         Verbose output
  --json            Output as JSON
```

### Configuration (`.promptsmith/config.yaml`)

```yaml
version: 1

project:
  name: my-ai-app
  id: proj_abc123

prompts_dir: ./prompts
tests_dir: ./tests
benchmarks_dir: ./benchmarks

defaults:
  model: gpt-4o
  temperature: 0.7

providers:
  openai:
    api_key_env: OPENAI_API_KEY
  anthropic:
    api_key_env: ANTHROPIC_API_KEY
  google:
    api_key_env: GOOGLE_API_KEY

sync:
  remote: https://app.promptsmith.dev
  auto_push: false
```

---

## Web Application Pages

### 1. Dashboard (`/`)
- Project overview
- Recent activity feed
- Quick stats (prompts, tests, benchmarks)
- Prompt health indicators

### 2. Prompts (`/prompts`)
- List all prompts with search/filter
- Version badges, last modified, test status
- Quick actions: edit, test, benchmark

### 3. Prompt Detail (`/prompts/[id]`)
- Version history (commit graph)
- Current content with syntax highlighting
- Variables panel
- Associated tests and benchmarks
- Quick diff between versions

### 4. Editor (`/prompts/[id]/edit`)
- Monaco-based prompt editor
- Live variable extraction
- Token counter
- Test runner sidebar
- Save as new version

### 5. Diff View (`/prompts/[id]/diff`)
- Version selector (dropdown or URL params)
- Side-by-side or unified diff
- Annotation threads
- Impact analysis

### 6. Tests (`/tests`)
- All test suites
- Recent run results
- Flaky test detection
- Coverage metrics

### 7. Test Detail (`/tests/[id]`)
- Individual test cases
- Run history
- Failure analysis
- Edit test definitions

### 8. Benchmarks (`/benchmarks`)
- All benchmark configurations
- Latest results comparison
- Model recommendation

### 9. Benchmark Detail (`/benchmarks/[id]`)
- Interactive charts
- Per-model breakdown
- Cost analysis
- Export reports

### 10. Settings (`/settings`)
- API key management
- Team members
- Integrations (GitHub, CI)
- Billing (if applicable)

---

## API Endpoints

### Prompts
```
GET    /api/prompts                 List all prompts
POST   /api/prompts                 Create prompt
GET    /api/prompts/:id             Get prompt
PUT    /api/prompts/:id             Update prompt
DELETE /api/prompts/:id             Delete prompt
GET    /api/prompts/:id/versions    List versions
POST   /api/prompts/:id/versions    Create version
GET    /api/prompts/:id/diff        Get diff between versions
```

### Tests
```
GET    /api/tests                   List test suites
POST   /api/tests                   Create test suite
GET    /api/tests/:id               Get test suite
POST   /api/tests/:id/run           Execute tests
GET    /api/tests/:id/runs          List test runs
GET    /api/tests/:id/runs/:runId   Get run results
```

### Benchmarks
```
GET    /api/benchmarks              List benchmarks
POST   /api/benchmarks              Create benchmark
GET    /api/benchmarks/:id          Get benchmark
POST   /api/benchmarks/:id/run      Execute benchmark
GET    /api/benchmarks/:id/runs     List benchmark runs
```

### Generation
```
POST   /api/generate/variations     Generate prompt variations
POST   /api/generate/compress       Compress prompt
POST   /api/generate/expand         Expand prompt
```

---

## LLM Integration

### Supported Providers

| Provider | Models | Status |
|----------|--------|--------|
| OpenAI | GPT-4o, GPT-4o-mini, GPT-4-turbo, o1-preview | P0 |
| Anthropic | Claude Opus/Sonnet/Haiku (3.5, 4) | P0 |
| Google | Gemini 1.5 Pro/Flash, Gemini 2.0 | P1 |
| Mistral | Large, Medium, Small | P1 |
| Groq | Llama 3.1, Mixtral | P2 |
| AWS Bedrock | Claude, Titan | P2 |
| Azure OpenAI | GPT-4 variants | P2 |
| Local | Ollama, LM Studio | P2 |

### Unified Interface

```typescript
interface LLMProvider {
  name: string;
  models: Model[];
  
  complete(request: CompletionRequest): Promise<CompletionResponse>;
  stream(request: CompletionRequest): AsyncIterable<StreamChunk>;
  
  estimateTokens(text: string, model: string): number;
  estimateCost(tokens: number, model: string): number;
}

interface CompletionRequest {
  prompt: string;
  model: string;
  temperature?: number;
  maxTokens?: number;
  stopSequences?: string[];
  variables?: Record<string, string>;
}

interface CompletionResponse {
  content: string;
  model: string;
  usage: {
    promptTokens: number;
    completionTokens: number;
    totalTokens: number;
  };
  latencyMs: number;
  cost: number;
}
```

---

## Milestones & Timeline

### Phase 1: Foundation (Weeks 1-4)
- [ ] Project scaffolding (Next.js, database schema)
- [ ] Basic CRUD for prompts and versions
- [ ] CLI skeleton with `init`, `add`, `commit`
- [ ] Simple web UI for prompt management

### Phase 2: Version Control (Weeks 5-8)
- [ ] Full versioning with branching
- [ ] Diff algorithm and visualization
- [ ] Tags and environment management
- [ ] CLI commands: `diff`, `log`, `tag`, `checkout`

### Phase 3: Testing (Weeks 9-12)
- [ ] Test suite definition format
- [ ] Assertion types implementation
- [ ] Test runner (CLI and web)
- [ ] CI integration (GitHub Actions)

### Phase 4: LLM Features (Weeks 13-16)
- [ ] Multi-provider integration
- [ ] Variation generation
- [ ] Benchmark execution
- [ ] Results visualization

### Phase 5: Polish & Launch (Weeks 17-20)
- [ ] Cloud sync functionality
- [ ] Team collaboration features
- [ ] Documentation site
- [ ] Public launch

---

## Success Metrics

| Metric | Target (6 months) |
|--------|-------------------|
| GitHub stars | 1,000+ |
| CLI downloads | 5,000+ |
| Active projects | 500+ |
| Prompts managed | 10,000+ |
| Test runs | 100,000+ |
| Benchmark executions | 50,000+ |

---

## Competitive Landscape

| Tool | Strengths | Weaknesses |
|------|-----------|------------|
| **Promptfoo** | Testing focus, good CLI | No versioning, limited UI |
| **LangSmith** | Full observability | Heavy, enterprise-focused |
| **Humanloop** | Good UI | Closed source, expensive |
| **Pezzo** | Open source | Limited features |
| **PromptSmith** | Full workflow, dev-first | New entrant |

### Differentiation
1. **Git-like mental model**: Familiar to developers
2. **CLI-first with great web UI**: Best of both worlds
3. **Open source core**: Community-driven, self-hostable
4. **Integrated workflow**: Version → Test → Benchmark in one tool

---

## Open Questions

1. **Pricing model**: Open source core + cloud tier? Or fully open source with enterprise features?
2. **Prompt format**: Invent new format or support existing (YAML, Jinja, Mustache)?
3. **Collaboration**: Real-time editing or async PR-style workflows?
4. **IDE extensions**: VS Code extension priority?

---

## Appendix

### Example Prompt File (`prompts/summarizer.prompt`)

```yaml
name: article-summarizer
description: Summarizes news articles into concise bullet points
model_hint: gpt-4o-mini  # Suggested model

variables:
  - name: article
    type: string
    required: true
  - name: max_points
    type: number
    default: 5
  - name: tone
    type: enum
    values: [formal, casual, technical]
    default: formal

---

You are a skilled editor who creates concise summaries.

Summarize the following article into {{max_points}} key bullet points.
Use a {{tone}} tone.

Article:
{{article}}

Summary:
```

### Example Test Output

```
$ promptsmith test summarizer

Running tests for summarizer@1.2.0...

  ✓ Short article summarization (1.2s)
  ✓ Long article summarization (2.8s)
  ✓ Handles empty input gracefully (0.4s)
  ✗ Technical article accuracy (1.9s)
    
    Assertion failed: semantic_similarity
    Expected similarity >= 0.85, got 0.72
    
    Diff:
    - Expected: "The study found a 40% improvement..."
    + Actual: "Researchers noted significant gains..."

  4 tests, 3 passed, 1 failed
  Total time: 6.3s
```

### Example Benchmark Report

```
$ promptsmith benchmark summarizer --models=gpt-4o,claude-sonnet,gemini-pro

Benchmark: summarizer@1.2.0
Dataset: 50 articles
Runs per model: 3

┌─────────────────┬──────────┬──────────┬────────────┬──────────┬─────────┐
│ Model           │ Latency  │ Tokens   │ Cost/1K    │ Quality  │ Score   │
│                 │ (p50)    │ (avg)    │ Requests   │ (0-100)  │         │
├─────────────────┼──────────┼──────────┼────────────┼──────────┼─────────┤
│ gpt-4o          │ 2.1s     │ 847      │ $0.42      │ 94       │ ★★★★★   │
│ claude-sonnet   │ 1.8s     │ 812      │ $0.38      │ 92       │ ★★★★☆   │
│ gemini-pro      │ 1.4s     │ 891      │ $0.22      │ 88       │ ★★★★☆   │
└─────────────────┴──────────┴──────────┴────────────┴──────────┴─────────┘

Recommendation: gpt-4o for quality, gemini-pro for cost efficiency
```

---

*Last updated: February 2026*
*Version: 1.0.0*
