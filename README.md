[![GitHub release](https://img.shields.io/github/release/UnitVectorY-Labs/ghprmerge.svg)](https://github.com/UnitVectorY-Labs/ghprmerge/releases/latest) [![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT) [![Active](https://img.shields.io/badge/Status-Active-green)](https://guide.unitvectorylabs.com/bestpractices/status/#active) [![Go Report Card](https://goreportcard.com/badge/github.com/UnitVectorY-Labs/ghprmerge)](https://goreportcard.com/report/github.com/UnitVectorY-Labs/ghprmerge) 

# ghprmerge

A command-line tool to automatically evaluate, merge, and optionally rebase GitHub pull requests sharing the same source branch across an organization.

Use case: merging automated dependency update pull requests (e.g., Dependabot) without requiring clicking through each repository individually.

## Quick Start

```bash
# Set your GitHub token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Optionally set the org as an environment variable (or use --org)
export GITHUB_ORG=myorg

# Check version
ghprmerge --version

# Show help with subcommand descriptions
ghprmerge --help

# Rebase out-of-date branches
ghprmerge rebase --org myorg --source-branch dependabot/

# Merge ready PRs (that are already up-to-date)
ghprmerge merge --org myorg --source-branch dependabot/

# PRs with no checks configured are allowed; pending checks still block merging

# Merge PRs even if behind (skip rebase requirement)
ghprmerge merge --org myorg --source-branch dependabot/ --skip-rebase

# Match multiple source branches
ghprmerge merge --org myorg --source-branch dependabot/ --source-branch feature/

# Filter by author (e.g. only Dependabot PRs opened by the app)
ghprmerge merge --author 'dependabot[bot]' --org myorg --source-branch dependabot/

# Use --confirm to review pending actions before taking action
ghprmerge rebase --org myorg --source-branch dependabot/ --confirm

# Stream all repo results as they are scanned
ghprmerge merge --verbose --org myorg --source-branch dependabot/

# Disable colored output
ghprmerge merge --no-color --org myorg --source-branch dependabot/

# Report mode: group open PRs by source branch
ghprmerge report --org myorg

# Report with prefix filter and JSON output
ghprmerge report --json --org myorg --source-branch-prefix dependabot/

# Report with minimum group size
ghprmerge report --org myorg --min-group-size 3
```

## Documentation

- [Purpose & Philosophy](docs/README.md) - Design goals and safety model
- [Usage Guide](docs/USAGE.md) - Complete command-line reference
- [Examples](docs/EXAMPLES.md) - Practical workflows
- [Merge Command](docs/MERGE.md) - Merge subcommand details
- [Rebase Command](docs/REBASE.md) - Rebase subcommand details
- [Report Command](docs/REPORT.md) - Report subcommand details
- [Installation](docs/INSTALL.md) - Installation instructions

## License

MIT License - see [LICENSE](LICENSE) for details.
