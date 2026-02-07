# PromptSmith

**The GitHub Copilot for Prompt Engineering**

PromptSmith brings software engineering best practices to prompt engineering. Version, test, iterate, and benchmark your LLM prompts with the same rigor you apply to code.

## Features

- **Version Control** — Git-like versioning with semantic versions (`prompt@1.2.3`)
- **Prompt Parsing** — YAML frontmatter + Mustache templates
- **Secret Scanning** — Detects API keys and credentials before commit
- **Testing** — Define test suites with 16+ assertion types, snapshot testing, flaky detection
- **Benchmarking** — Compare prompts across OpenAI and Anthropic models, result comparison
- **AI Generation** — Generate variations, compress, or expand prompts with LLMs
- **Cloud Sync** — Push and pull prompts to/from remote for collaboration
- **Web Dashboard** — CodeMirror editor, inline diff comments, export reports, recommendation cards
- **Documentation** — VitePress docs site with CLI, API, and Web UI reference

## Installation

```bash
# Build from source
cd cli
go build -o promptsmith .

# Add to PATH (optional)
sudo mv promptsmith /usr/local/bin/
```

## Quick Start

```bash
# Initialize a project
promptsmith init my-ai-app

# Create a prompt file
cat > prompts/summarizer.prompt << 'EOF'
---
name: article-summarizer
description: Summarizes articles into bullet points
model_hint: gpt-4o-mini
variables:
  - name: article
    type: string
    required: true
  - name: max_points
    type: number
    default: 5
---

Summarize this article into {{max_points}} bullet points:

{{article}}
EOF

# Track the prompt
promptsmith add prompts/summarizer.prompt

# Commit a version
promptsmith commit -m "Initial summarizer prompt"

# View history
promptsmith log
```

## Commands

| Command | Description |
|---------|-------------|
| `promptsmith init [name]` | Initialize a new project |
| `promptsmith add <file>` | Track a prompt file |
| `promptsmith remove <prompt>` | Stop tracking a prompt |
| `promptsmith commit -m "msg"` | Create new version for changed prompts |
| `promptsmith status` | Show project status and uncommitted changes |
| `promptsmith list` | List all tracked prompts with versions |
| `promptsmith show <prompt>` | Display prompt details and content |
| `promptsmith log` | Show version history |
| `promptsmith log -p <name>` | Show history for specific prompt |
| `promptsmith diff <prompt> [v1] [v2]` | Compare versions (unified diff) |
| `promptsmith tag <prompt> <name> [ver]` | Create named version tag |
| `promptsmith tag <prompt> --list` | List all tags |
| `promptsmith checkout <prompt> <ref>` | Switch to version or tag |
| `promptsmith test [files...]` | Run test suites |
| `promptsmith test --watch` | Watch mode - re-run tests on file changes |
| `promptsmith test --update-snapshots` | Update snapshot assertions with current output |
| `promptsmith benchmark [files...]` | Run model benchmarks |
| `promptsmith benchmark compare <f1> <f2>` | Compare two benchmark result files |
| `promptsmith generate <prompt>` | Generate prompt variations with AI |
| `promptsmith config` | View/modify project configuration |
| `promptsmith serve` | Start API server for web UI integration |
| `promptsmith login` | Authenticate with PromptSmith cloud |
| `promptsmith logout` | Log out from PromptSmith cloud |
| `promptsmith whoami` | Show current user info |
| `promptsmith push` | Sync local changes to cloud |
| `promptsmith pull` | Fetch latest from cloud |

Version references support `HEAD`, `HEAD~1`, `HEAD~2`, etc.

### Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `-V, --verbose` | Verbose output |

## Prompt File Format

Prompts use YAML frontmatter with Mustache templates:

```yaml
---
name: my-prompt
description: What this prompt does
model_hint: gpt-4o

variables:
  - name: input
    type: string
    required: true
  - name: style
    type: enum
    values: [formal, casual]
    default: formal
---

Your prompt content here with {{input}} and {{style}} variables.
```

### Variable Types

- `string` — Text input
- `number` — Numeric value
- `enum` — One of predefined values

## Project Structure

```
my-project/
├── .promptsmith/
│   ├── config.yaml      # Project configuration
│   └── promptsmith.db   # Version database (gitignored)
├── prompts/             # Your prompt files
├── tests/               # Test suite definitions
└── benchmarks/          # Benchmark configurations
```

## Testing

Define test suites in YAML to validate your prompts:

```yaml
# tests/summarizer.test.yaml
name: summarizer-tests
prompt: summarizer
tests:
  - name: basic-output
    inputs:
      article: "AI is transforming industries."
      max_points: 3
    assertions:
      - type: not_empty
      - type: max_length
        value: 500
      - type: min_lines
        value: 3

  - name: json-format
    inputs:
      article: "Test article"
    assertions:
      - type: json_valid
      - type: json_path
        path: "summary"
```

Run tests:

```bash
promptsmith test                    # Run all tests in tests/
promptsmith test --filter "basic"   # Run matching tests
promptsmith test --version 1.0.0    # Test specific version
promptsmith test --live             # Run with real LLM (requires API key)
promptsmith test --live --model gpt-4o  # Use specific model
```

### Assertion Types

| Type | Description |
|------|-------------|
| `contains` | Output contains value |
| `not_contains` | Output doesn't contain value |
| `equals` | Output matches exactly |
| `matches` | Output matches regex |
| `starts_with` | Output starts with value |
| `ends_with` | Output ends with value |
| `min_length` | Minimum character count |
| `max_length` | Maximum character count |
| `not_empty` | Output is not empty |
| `json_valid` | Output is valid JSON |
| `json_path` | JSONPath query exists or matches |
| `line_count` | Exact line count |
| `min_lines` | Minimum line count |
| `max_lines` | Maximum line count |
| `word_count` | Exact word count |
| `snapshot` | Compare against stored `expected_output` |

## Benchmarking

Compare prompt performance across different LLM providers:

```yaml
# benchmarks/summarizer.bench.yaml
name: summarizer-benchmark
prompt: summarizer
models:
  - gpt-4o
  - gpt-4o-mini
  - claude-sonnet
runs_per_model: 5
```

Run benchmarks:

```bash
promptsmith benchmark                              # Run all benchmarks
promptsmith benchmark --models gpt-4o,claude-sonnet
promptsmith benchmark --runs 10                    # 10 runs per model
promptsmith benchmark -o results.json              # Save results
promptsmith benchmark compare base.json latest.json # Compare results
```

Benchmark output shows latency percentiles (p50, p99), token usage, cost per request, and recommendations for best speed/cost models. The `compare` subcommand shows a color-coded delta table between two result files.

### Supported Models

**OpenAI**: gpt-4o, gpt-4o-mini, gpt-4-turbo, o1, o1-mini

**Anthropic**: claude-sonnet, claude-haiku, claude-opus (and dated versions)

Set API keys via environment variables:
- `OPENAI_API_KEY`
- `ANTHROPIC_API_KEY`

## Prompt Generation

Generate prompt variations using AI:

```bash
promptsmith generate summarizer                    # Generate 3 variations
promptsmith generate summarizer --count 5          # Generate 5 variations
promptsmith generate summarizer --type compress    # Compress the prompt
promptsmith generate summarizer --type expand      # Expand with more detail
promptsmith generate summarizer --goal "be concise"
promptsmith generate summarizer --model claude-sonnet
```

### Generation Types

| Type | Description |
|------|-------------|
| `variations` | Create alternative versions with different approaches |
| `compress` | Reduce token count while preserving functionality |
| `expand` | Add more detail, examples, and edge case handling |
| `rephrase` | Reword while keeping the same meaning |

## Secret Scanning

PromptSmith warns you about potential secrets before committing:

```
⚠ Potential secrets detected:
  Line 7: OpenAI API Key - sk-1...efgh

Consider removing sensitive data before committing.
```

Detected patterns:
- AWS Access/Secret Keys
- GitHub/GitLab Tokens
- OpenAI, Anthropic, Google API Keys
- Slack, Stripe Tokens
- Private Keys
- Database URLs
- Generic secrets (`api_key=`, `password=`, etc.)

## Web UI

PromptSmith includes a web interface for browsing prompts and managing tests/benchmarks.

```bash
# Start the API server (in your project directory)
promptsmith serve  # Runs on http://localhost:8080

# Start the web UI
cd web
npm install
npm run dev        # Runs on http://localhost:8081
```

Features:
- **Dashboard** — Project stats (prompts, test suites, test cases, benchmarks)
- **Prompt list** — Version badges, search/filter, create new prompts
- **Prompt detail** — Tabbed view: content, history, diff with inline comments, change impact preview
- **Prompt editor** — CodeMirror 6 with syntax highlighting, `{{variable}}` detection, dark theme
- **Version history** — Commit messages, tag management (add/remove)
- **Diff viewer** — Unified diff with per-line comments, change impact preview (affected tests/benchmarks)
- **Tests page** — Browse test suites, run tests, flaky test detection, export JSON/CSV
- **Benchmarks page** — Browse benchmarks, run and compare, recommendation cards (best overall/throughput/budget), export JSON/CSV
- **Settings** — Project info, LLM provider config, team/sync configuration
- **AI generation** — Generate prompt variations, compress, expand, rephrase

### API Server

The `serve` command starts a REST API for integration:

```bash
promptsmith serve              # Default: http://localhost:8080
promptsmith serve --port 3000  # Custom port
```

**Endpoints:**
- `GET  /api/project` — Project info
- `GET  /api/config/sync` — Sync configuration
- `GET  /api/prompts` — List all prompts
- `POST /api/prompts` — Create prompt
- `GET  /api/prompts/:name` — Get prompt details
- `PUT  /api/prompts/:name` — Update prompt metadata
- `DELETE /api/prompts/:name` — Delete prompt
- `GET  /api/prompts/:name/versions` — List versions
- `POST /api/prompts/:name/versions` — Create new version
- `GET  /api/prompts/:name/diff?v1=X&v2=Y` — Version diff
- `POST /api/prompts/:name/tags` — Create tag
- `DELETE /api/prompts/:name/tags/:tag` — Delete tag
- `GET  /api/prompts/:name/comments` — List inline comments
- `POST /api/prompts/:name/comments` — Create comment
- `DELETE /api/comments/:id` — Delete comment
- `GET  /api/tests` — List test suites
- `POST /api/tests` — Create test suite
- `GET  /api/tests/:name` — Get test suite
- `POST /api/tests/:name/run` — Run test suite
- `GET  /api/tests/:name/runs` — Test run history
- `GET  /api/tests/:name/runs/:runId` — Get test run
- `GET  /api/benchmarks` — List benchmarks
- `POST /api/benchmarks` — Create benchmark suite
- `GET  /api/benchmarks/:name` — Get benchmark
- `POST /api/benchmarks/:name/run` — Run benchmark
- `GET  /api/benchmarks/:name/runs` — Benchmark run history
- `POST /api/generate` — Generate prompt variations
- `POST /api/generate/compress` — Compress prompt
- `POST /api/generate/expand` — Expand prompt

## Cloud Sync

Sync your prompts with the PromptSmith cloud for backup and team collaboration.

### Authentication

```bash
# Interactive login
promptsmith login

# Token-based login (for CI/CD)
promptsmith login --token <your-token>

# Or use environment variable
export PROMPTSMITH_TOKEN=<your-token>

# Log out
promptsmith logout
```

### Syncing

```bash
# Push local changes to cloud
promptsmith push

# Pull remote changes
promptsmith pull

# Force push (overwrite remote conflicts)
promptsmith push --force

# Force pull (overwrite local changes)
promptsmith pull --force
```

### Configuration

```bash
# Set remote URL (defaults to https://api.promptsmith.dev)
promptsmith config sync.remote https://your-server.com

# Enable auto-push on commit
promptsmith config sync.auto_push true

# Set team for collaboration
promptsmith config sync.team my-team
```

## Documentation

Full documentation is available in the `docs/` directory, powered by VitePress:

```bash
cd docs
npm install
npm run docs:dev     # Dev server
npm run docs:build   # Production build
```

Pages: [Getting Started](docs/getting-started.md) | [CLI Reference](docs/cli-reference.md) | [Web UI](docs/web-ui.md) | [API Reference](docs/api-reference.md) | [Contributing](docs/contributing.md)

## Roadmap

- [x] **Phase 1**: CLI foundation, versioning, parsing
- [x] **Phase 2**: Diff, tags, web UI scaffolding
- [x] **Phase 3**: Testing framework with 15+ assertion types
- [x] **Phase 4**: Multi-model benchmarking, AI generation, live testing
- [x] **Phase 5**: Cloud sync, collaboration
- [x] **Phase 6**: Full web UI — editor, tests, benchmarks, settings, dashboard
- [x] **Phase 7**: CRUD completion, tag management, confirmations
- [x] **Phase 8**: CodeMirror editor, inline comments, snapshot testing, benchmark compare, flaky detection, export reports, VitePress docs

## License

MIT
