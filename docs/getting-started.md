# Getting Started

## Installation

```bash
go install github.com/promptsmith/cli@latest
```

Or build from source:

```bash
git clone https://github.com/promptsmith/promptsmith.git
cd promptsmith/cli
go build -o promptsmith .
```

## Initialize a Project

```bash
mkdir my-prompts && cd my-prompts
promptsmith init
```

This creates a `.promptsmith/` directory with a SQLite database and config file.

## Create Your First Prompt

```bash
promptsmith add summarizer --description "Summarize articles"
```

Edit the prompt file at `prompts/summarizer.prompt`:

```
Summarize the following article in {{.max_points}} bullet points:

{{.article}}
```

## Commit a Version

```bash
promptsmith commit summarizer -m "Initial summarizer prompt"
```

## View History

```bash
promptsmith log summarizer
```

## Run Tests

Create a test file at `tests/summarizer.test.yaml`:

```yaml
name: summarizer-tests
prompt: summarizer
tests:
  - name: basic-summary
    inputs:
      article: "AI is transforming many industries..."
      max_points: 3
    assertions:
      - type: not_empty
      - type: max_length
        value: 500
```

Then run:

```bash
promptsmith test
```

## Start the Web UI

```bash
promptsmith serve
```

Open `http://localhost:8080` in your browser to access the dashboard.
