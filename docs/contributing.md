# Contributing

## Project Structure

```
promptsmith/
  cli/           # Go CLI and API server
    cmd/         # Cobra commands
    internal/
      api/       # HTTP server
      db/        # SQLite database layer
      testing/   # Test runner and assertions
      benchmark/ # Benchmark runner and providers
      generator/ # AI-powered prompt generation
      prompt/    # Prompt parsing
      scanner/   # File scanner
      sync/      # Team sync
  web/           # React + TypeScript frontend
    src/
      pages/     # Page components
      components/# Shared components
  docs/          # VitePress documentation
```

## Development Setup

### CLI

```bash
cd cli
go build -o promptsmith .
go test ./...
```

### Web

```bash
cd web
npm install
npx vite --port 8081    # Dev server
npm run test:run        # Run tests
```

### Docs

```bash
cd docs
npx vitepress dev       # Dev server
npx vitepress build     # Production build
```

## Guidelines

- Make frequent, atomic commits
- Run tests before pushing (`go test ./...` and `npm run test:run`)
- Follow existing code patterns (CSS Modules, CSS variables, vi.mock patterns)
- Use TypeScript strict mode in the web project
