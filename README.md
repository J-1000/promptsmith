# PromptSmith

**The GitHub Copilot for Prompt Engineering**

PromptSmith brings software engineering best practices to prompt engineering. Version, test, iterate, and benchmark your LLM prompts with the same rigor you apply to code.

## Features

- **Version Control** — Git-like versioning with semantic versions (`prompt@1.2.3`)
- **Prompt Parsing** — YAML frontmatter + Mustache templates
- **Secret Scanning** — Detects API keys and credentials before commit
- **Testing** — Define test suites with 15+ assertion types
- **Benchmarking** — Compare prompts across OpenAI and Anthropic models
- **AI Generation** — Generate variations, compress, or expand prompts with LLMs

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
| `promptsmith commit -m "msg"` | Create new version for changed prompts |
| `promptsmith status` | Show project status and uncommitted changes |
| `promptsmith list` | List all tracked prompts with versions |
| `promptsmith log` | Show version history |
| `promptsmith log -p <name>` | Show history for specific prompt |
| `promptsmith diff <prompt> [v1] [v2]` | Compare versions (unified diff) |
| `promptsmith tag <prompt> <name> [ver]` | Create named version tag |
| `promptsmith tag <prompt> --list` | List all tags |
| `promptsmith checkout <prompt> <ref>` | Switch to version or tag |
| `promptsmith test [files...]` | Run test suites |
| `promptsmith test --watch` | Watch mode - re-run tests on file changes |
| `promptsmith benchmark [files...]` | Run model benchmarks |
| `promptsmith generate <prompt>` | Generate prompt variations with AI |
| `promptsmith config` | View/modify project configuration |
| `promptsmith serve` | Start API server for web UI integration |

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
```

Benchmark output shows latency percentiles (p50, p99), token usage, cost per request, and recommendations for best speed/cost models.

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
cd web
npm install
npm run dev    # http://localhost:5173
```

Features:
- Prompt list with version badges and tags
- Version history with commit messages
- Unified diff viewer for comparing versions
- Test results dashboard with pass/fail indicators
- Benchmark results with model comparison table
- AI-powered prompt variation generator

## Roadmap

- [x] **Phase 1**: CLI foundation, versioning, parsing
- [x] **Phase 2**: Diff, tags, web UI scaffolding
- [x] **Phase 3**: Testing framework with 15+ assertion types
- [x] **Phase 4**: Multi-model benchmarking, AI generation, live testing
- [ ] **Phase 5**: Cloud sync, collaboration

## License

MIT
