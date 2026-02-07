# API Reference

The PromptSmith API server runs on port 8080 by default.

## Project

### `GET /api/project`

Returns project metadata.

```json
{ "id": "abc123", "name": "my-project" }
```

## Prompts

### `GET /api/prompts`

List all prompts.

### `GET /api/prompts/:name`

Get a single prompt by name.

### `POST /api/prompts`

Create a new prompt.

```json
{ "name": "summarizer", "description": "Summarize articles", "content": "optional initial content" }
```

### `PUT /api/prompts/:name`

Update prompt metadata.

```json
{ "name": "new-name", "description": "updated description" }
```

### `DELETE /api/prompts/:name`

Delete a prompt.

## Versions

### `GET /api/prompts/:name/versions`

List all versions of a prompt.

### `POST /api/prompts/:name/versions`

Create a new version.

```json
{ "content": "prompt content here", "commit_message": "describe the change" }
```

### `GET /api/prompts/:name/diff?v1=1.0.0&v2=1.1.0`

Get a unified diff between two versions.

## Tags

### `POST /api/prompts/:name/tags`

Create a tag for a version.

```json
{ "name": "production", "version_id": "version-uuid" }
```

### `DELETE /api/prompts/:name/tags/:tagName`

Delete a tag.

## Comments

### `GET /api/prompts/:name/comments`

List comments on a prompt.

### `POST /api/prompts/:name/comments`

Create an inline comment.

```json
{ "version_id": "version-uuid", "line_number": 5, "content": "This could be clearer" }
```

### `DELETE /api/comments/:id`

Delete a comment.

## Tests

### `GET /api/tests`

List all test suites.

### `GET /api/tests/:name`

Get a test suite by name.

### `POST /api/tests`

Create a new test suite.

```json
{ "name": "my-tests", "prompt": "summarizer", "description": "optional" }
```

### `POST /api/tests/:name/run`

Run a test suite. Returns `SuiteResult`.

### `GET /api/tests/:name/runs`

List previous test runs.

### `GET /api/tests/:name/runs/:runId`

Get a specific test run.

## Benchmarks

### `GET /api/benchmarks`

List all benchmark suites.

### `GET /api/benchmarks/:name`

Get a benchmark suite by name.

### `POST /api/benchmarks`

Create a new benchmark suite.

```json
{ "name": "my-bench", "prompt": "summarizer", "models": ["gpt-4o", "claude-sonnet"], "runs_per_model": 5 }
```

### `POST /api/benchmarks/:name/run`

Run a benchmark. Returns `BenchmarkResult`.

### `GET /api/benchmarks/:name/runs`

List previous benchmark runs.

## Generate

### `POST /api/generate`

Generate prompt variations.

```json
{ "type": "variations", "prompt": "content", "count": 3, "goal": "optional", "model": "optional" }
```

### `POST /api/generate/compress`

Compress a prompt.

```json
{ "prompt": "content", "goal": "optional", "model": "optional" }
```

### `POST /api/generate/expand`

Expand a prompt with more detail.

```json
{ "prompt": "content", "goal": "optional", "model": "optional" }
```

## Configuration

### `GET /api/config/sync`

Get sync configuration.

```json
{ "team": "my-team", "remote": "https://...", "auto_push": true, "status": "configured" }
```
