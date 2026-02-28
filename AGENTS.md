# AGENTS.md

## Project: ghprmerge

This is a Go CLI application. When making changes, follow these conventions:

### Documentation Sync

Documentation in `docs/` and `README.md` must be kept in sync with the implementation. When modifying CLI flags, output behavior, or execution flow, update:

- `docs/USAGE.md` — flags table, flag combinations, output format, and behavioral descriptions
- `docs/EXAMPLES.md` — example commands demonstrating new or changed flags
- `docs/README.md` — execution flow overview
- `README.md` — quick start examples

### Code Style

- Use `gofmt` to format all Go code before committing.
- Run tests with `go test ./...` and build with `go build -v`.
