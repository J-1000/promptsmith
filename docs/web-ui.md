# Web UI

The PromptSmith web dashboard provides a visual interface for managing prompts, viewing diffs, running tests, and analyzing benchmarks.

## Starting the UI

```bash
promptsmith serve
```

The server starts on port 8080 by default. The web UI is available at `http://localhost:8080`.

For development, the Vite dev server runs on port 8081:

```bash
cd web && npx vite --port 8081
```

## Pages

### Home

Overview of all prompts in the project with quick stats and a search/filter interface. Create new prompts directly from the UI.

### Prompt Detail

View a prompt's content, version history, and diffs between versions. Features include:

- **Editor tab**: Edit prompt content with CodeMirror (syntax highlighting, `{{variable}}` detection)
- **History tab**: Browse version timeline
- **Diff tab**: Compare any two versions side-by-side with inline comments and change impact preview
- **Tag management**: Create and delete version tags

### Tests

List all test suites with pass/fail status. Run tests from the UI and view detailed results including assertion outcomes. Flaky tests are flagged automatically.

### Test Results

Drill into individual test runs with per-assertion details. Export results as JSON or CSV.

### Benchmarks

List benchmark suites and trigger runs. View model comparison tables.

### Benchmark Results

Detailed results with latency, cost, token usage, and error rates per model. Includes recommendation cards (best overall, best throughput, best budget). Export as JSON or CSV.

### Editor

Full-featured prompt editor with CodeMirror 6, markdown support, and live variable highlighting.

### Generate

AI-powered prompt variations, compression, expansion, and rephrasing.

### Settings

Project configuration, sync settings, and team information.
