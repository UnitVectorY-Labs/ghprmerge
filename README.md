# ghprmerge

Automatically finds and safely merges large numbers of similar GitHub pull requests across an organization by verifying all checks pass and branches are fully up to date, with optional automated branch updates.

## Overview

`ghprmerge` is a small, reliable command-line application written in Go that can safely and efficiently merge large numbers of similar GitHub pull requests across an organization. The primary use case is merging automated dependency update pull requests, such as those created by Dependabot, without requiring repositories to be checked out locally.

## Key Features

- **Safe by default**: Dry-run mode is enabled by default
- **Strict readiness checks**: Only merges PRs where all checks pass and branches are up to date
- **No local checkouts**: All operations are performed via the GitHub API
- **Flexible filtering**: Filter by organization, repository, and branch patterns
- **Automated branch updates**: Optionally update out-of-date branches before merging
- **Dependabot support**: Special handling for Dependabot branches via rebase comments

## Quick Start

```bash
# Set your GitHub token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Preview what would be merged (dry-run is default)
ghprmerge --org myorg --source-branch dependabot/

# Actually merge the PRs
ghprmerge --org myorg --source-branch dependabot/ --dry-run=false
```

## Readiness Requirements

A pull request is considered ready to merge only if ALL of the following are true:

1. All check runs have a successful conclusion
2. All commit statuses are successful
3. No merge conflicts
4. Branch is fully up to date with the default branch

## Installation

See [docs/INSTALL.md](docs/INSTALL.md) for detailed installation instructions.

### Quick Install

```bash
go install github.com/UnitVectorY-Labs/ghprmerge/cmd/ghprmerge@latest
```

## Documentation

- [Usage Guide](docs/USAGE.md) - Complete command-line reference
- [Examples](docs/EXAMPLES.md) - Practical example commands
- [Installation](docs/INSTALL.md) - Installation instructions

## Command Line Options

| Flag | Default | Description |
|------|---------|-------------|
| `--org` | - | GitHub organization to scan (required) |
| `--source-branch` | - | Branch pattern to match (required) |
| `--dry-run` | `true` | Preview mode, no changes made |
| `--rebase` | `false` | Update out-of-date branches |
| `--repo` | - | Limit to specific repos (repeatable) |
| `--limit` | `0` | Max PRs to merge (0 = unlimited) |
| `--json` | `false` | JSON output format |

## License

MIT License - see [LICENSE](LICENSE) for details.
