# 🤝 Contributing to Dockit Desktop

First off, thank you for considering contributing to Dockit Desktop! It's people like you that make this tool better for everyone. This guide explains how to set up the project, make changes safely, and submit high-quality pull requests.

## 🔗 Quick Links
- **Code of Conduct**: Not present yet (be excellent to each other!)
- **License**: See [`LICENSE`](LICENSE)

## 💡 Ways to Contribute
- 🐛 **Bug reports**: Provide clear reproduction steps, expected vs actual behavior, and logs.
- ✨ **Feature requests**: Describe the problem, your proposed solution, and any alternatives.
- 📚 **Documentation**: Fix typos, clarify setup, add examples, or improve screenshots.
- 💻 **Code changes**: Fix bugs, refactor existing systems, or add awesome new features.

---

## 🛠️ Development Setup

### Prerequisites
- **Go 1.25+**
- **Node.js 18+** & npm
- **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **Docker Engine** or Docker Desktop (required for Docker features)
- **PostgreSQL server** (optional, for DB Manager testing)

### Install Dependencies
```bash
npm install --prefix frontend
```

### Run in Development
```bash
wails dev
```
> *This starts Vite and the Wails dev server. Use `http://localhost:34115` to access the devtools bridge.*

---

## 📁 Project Layout
- `main.go` - App entrypoint
- `bindings/` - Wails bindings used by the frontend
- `internal/` - Domain, ports, usecases, and infrastructure adapters
- `frontend/` - React UI built with Vite
- `frontend/wailsjs/` - Auto-generated bindings (**do not edit manually**)

---

## 📏 Coding Guidelines

### Go
- Use `gofmt` on all Go files before committing.
- Keep usecase logic isolated in `internal/usecase/`.
- Keep external integrations (DB, Docker, etc.) in `internal/infrastructure/`.
- Keep interfaces well-defined in `internal/ports/`.

### Frontend
- Use **TypeScript** for all new code.
- Follow existing UI styles (glass-card, badges, buttons).
- Keep pages organized in `frontend/src/pages/`.
- Keep layout and navigation components in `frontend/src/layouts/`.
- Avoid inline refactors that are unrelated to your current change.

---

## ✅ Running Checks
There is no formal test suite yet, but please ensure your code is clean before submitting:

```bash
# Format Go code
gofmt -w ./...

# Build frontend to check for TS/Lint errors
npm run build --prefix frontend
```

If possible on your platform, also verify the complete Wails build:
```bash
wails build
```

---

## 🚀 Submitting a Pull Request
1. **Fork** the repository and create your feature branch (`git checkout -b feature/amazing-feature`).
2. Make **focused, minimal changes** that directly address the issue.
3. Update **docs and screenshots** if your change affects the UI.
4. Ensure all checks in the "Running Checks" section pass.
5. Open a **PR** with a clear summary, and include any relevant logs or screenshots.

### 📝 Commit Message Suggestions
We prefer clear and descriptive commit messages:
- `fix: postgres connection parsing logic`
- `feat: add env variable search bar`
- `docs: update README screenshots with new UI`
- `refactor: simplify docker container polling`

---

## 🔒 Reporting Security Issues
If you discover a security issue, please **avoid public disclosure**. Open a private vulnerability report on GitHub or contact the maintainers directly.
