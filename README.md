# ghprmerge

A command-line tool to safely merge large numbers of similar GitHub pull requests across an organization.

Primary use case: merging automated dependency update pull requests (e.g., Dependabot) without requiring repositories to be checked out locally.

## Quick Start

```bash
# Set your GitHub token
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Analyze what would be merged (default mode)
ghprmerge --org myorg --source-branch dependabot/

# Rebase out-of-date branches
ghprmerge --org myorg --source-branch dependabot/ --rebase

# Merge ready PRs
ghprmerge --org myorg --source-branch dependabot/ --merge

# Rebase and merge in one run
ghprmerge --org myorg --source-branch dependabot/ --rebase --merge
```

## Documentation

- [Purpose & Philosophy](docs/README.md) - Design goals and safety model
- [Usage Guide](docs/USAGE.md) - Complete command-line reference
- [Examples](docs/EXAMPLES.md) - Practical workflows
- [Installation](docs/INSTALL.md) - Installation instructions

## License

MIT License - see [LICENSE](LICENSE) for details.
