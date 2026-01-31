# ghprmerge

A command-line tool to automatically evaluate, merge, and optionally rebase GitHub pull requests sharing the same source branch across an organization.

Use case: merging automated dependency update pull requests (e.g., Dependabot) without requiring clicking through each repository individually.

## Quick Start

```bash
# Set your GitHub token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Check version
ghprmerge --version

# Analyze what would be merged (default mode)
ghprmerge --org myorg --source-branch dependabot/

# Rebase out-of-date branches
ghprmerge --org myorg --source-branch dependabot/ --rebase

# Merge ready PRs (that are already up-to-date)
ghprmerge --org myorg --source-branch dependabot/ --merge

# Merge PRs even if behind (skip rebase requirement)
ghprmerge --org myorg --source-branch dependabot/ --merge --skip-rebase

# Use --confirm to review before taking actions
ghprmerge --org myorg --source-branch dependabot/ --rebase --confirm
```

## Documentation

- [Purpose & Philosophy](docs/README.md) - Design goals and safety model
- [Usage Guide](docs/USAGE.md) - Complete command-line reference
- [Examples](docs/EXAMPLES.md) - Practical workflows
- [Installation](docs/INSTALL.md) - Installation instructions

## License

MIT License - see [LICENSE](LICENSE) for details.
