# Contributing to Dockit Desktop

Thanks for contributing. This guide explains how to set up the project, make changes safely, and submit high quality pull requests.

## Quick Links
- Code of Conduct: Not present yet
- License: See `LICENSE`

## Ways to Contribute
- **Bug reports**: Provide clear reproduction steps, expected vs actual behavior, and logs
- **Feature requests**: Describe the problem, proposed solution, and alternatives
- **Documentation**: Fix typos, clarify setup, add examples, improve screenshots
- **Code changes**: Fix bugs, refactor, or add new features

## Development Setup
### Prerequisites
- Go 1.25+
- Node.js 18+ and npm
- Wails CLI: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- Docker Engine or Docker Desktop (for Docker features)
- PostgreSQL server (optional, for DB Manager testing)

### Install Dependencies
```bash
npm install --prefix frontend
```

### Run in Development
```bash
wails dev
```

This starts Vite and the Wails dev server. Use `http://localhost:34115` to access the devtools bridge.

## Project Layout
- `main.go` app entrypoint
- `bindings/` Wails bindings used by the frontend
- `internal/` domain, ports, usecases, infrastructure adapters
- `frontend/` React UI (Vite)
- `frontend/wailsjs/` auto-generated bindings (do not edit)

## Coding Guidelines
### Go
- Use `gofmt` on all Go files
- Keep usecase logic in `internal/usecase/`
- Keep external integrations in `internal/infrastructure/`
- Keep interfaces in `internal/ports/`

### Frontend
- Use TypeScript and follow existing UI styles (glass-card, badges, buttons)
- Keep pages in `frontend/src/pages/`
- Keep layout and navigation in `frontend/src/layouts/`
- Avoid inline refactors that are unrelated to the change

## Running Checks
There is no formal test suite yet. Please run:
```bash
gofmt -w ./...
npm run build --prefix frontend
```

If possible on your platform, also run:
```bash
wails build
```

## Submitting a Pull Request
1) Fork the repository and create a branch
2) Make focused, minimal changes
3) Update docs and screenshots when UI changes
4) Ensure checks in "Running Checks" pass
5) Open a PR with a clear summary and any relevant logs or screenshots

## Commit Message Suggestions
- `fix: postgres connection parsing`
- `feat: add env variable search`
- `docs: update README screenshots`

## Reporting Security Issues
If you discover a security issue, avoid public disclosure. Open a private report or contact the maintainer directly.
