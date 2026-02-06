# Contributing to PromptSmith

## Setup

### CLI (Go)

```bash
cd cli
go mod download
go build -o promptsmith .
```

### Web (React + TypeScript)

```bash
cd web
npm install
npx vite --port 8081
```

## Running Tests

### Go tests

```bash
cd cli
go test ./...
```

### Web tests

```bash
cd web
npm run test:run
```

## Development Workflow

1. Fork and clone the repo
2. Create a feature branch: `git checkout -b feat/my-feature`
3. Make changes with tests
4. Run all tests before committing
5. Submit a PR against `main`

## Code Style

- **Go**: `gofmt` formatting, standard Go idioms
- **TypeScript**: Strict mode, CSS Modules for styling
- **CSS**: Use CSS variables defined in `web/src/index.css`
- **Commits**: Atomic commits with descriptive messages

## Project Structure

```
cli/                  # Go CLI + API server
  internal/
    api/              # HTTP API handlers
    db/               # SQLite database layer
    testing/          # Test runner
    benchmark/        # Benchmark runner
    generator/        # AI generation
web/                  # React frontend
  src/
    components/       # Reusable components
    pages/            # Page components
    api.ts            # API client
```
