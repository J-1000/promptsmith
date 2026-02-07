# CLI Reference

## Global Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON |
| `-V, --verbose` | Verbose output |

## Commands

### `init`

Initialize a new PromptSmith project.

```bash
promptsmith init
```

### `add`

Add a new prompt to the project.

```bash
promptsmith add <name> [--description "desc"]
```

### `commit`

Commit the current version of a prompt.

```bash
promptsmith commit <name> -m "commit message"
```

### `log`

View version history for a prompt.

```bash
promptsmith log <name>
```

### `diff`

Show differences between two versions.

```bash
promptsmith diff <name> <v1> <v2>
```

### `show`

Display a prompt's content at a specific version.

```bash
promptsmith show <name> [--version <v>]
```

### `list`

List all prompts in the project.

```bash
promptsmith list
```

### `status`

Show the current state of prompts (modified, untracked).

```bash
promptsmith status
```

### `tag`

Manage version tags.

```bash
promptsmith tag <name> <tag-name> [--version <v>]
promptsmith tag <name> --delete <tag-name>
```

### `test`

Run test suites against prompts.

```bash
promptsmith test [suite-file...]
promptsmith test --filter "basic"
promptsmith test --version 1.0.0
promptsmith test --live --model gpt-4o
promptsmith test --watch
promptsmith test --update-snapshots
```

| Flag | Description |
|------|-------------|
| `-f, --filter` | Only run tests matching pattern |
| `-v, --version` | Test against specific version |
| `--live` | Run against real LLMs |
| `-m, --model` | Model for live testing (default: gpt-4o-mini) |
| `-w, --watch` | Re-run on file changes |
| `--update-snapshots` | Update snapshot assertions |
| `-o, --output` | Write results to JSON file |

### `benchmark`

Run benchmark suites to compare models.

```bash
promptsmith benchmark [suite-file...]
promptsmith benchmark --models gpt-4o,claude-sonnet
promptsmith benchmark --runs 10
promptsmith benchmark -o results.json
```

### `benchmark compare`

Compare two benchmark result files.

```bash
promptsmith benchmark compare baseline.json latest.json
```

### `generate`

Generate prompt variations using AI.

```bash
promptsmith generate <name> [--type variations|compress|expand|rephrase]
```

### `config`

View or set project configuration.

```bash
promptsmith config                    # Show all config
promptsmith config defaults.model     # Get specific key
promptsmith config defaults.model gpt-4o  # Set value
```

### `serve`

Start the API server and web UI.

```bash
promptsmith serve [--port 8080]
```
