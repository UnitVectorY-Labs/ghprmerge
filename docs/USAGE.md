# Usage

## Command Line Reference

```
ghprmerge [flags]
```

## Required Parameters

| Flag | Environment Variable | Description |
|------|---------------------|-------------|
| `--org` | `GITHUB_ORG` | The GitHub organization to scan |
| `--source-branch` | - | A string used to identify candidate pull requests by matching against the pull request head branch name |

## Optional Parameters

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | `true` | When enabled, no mutations are performed |
| `--rebase` | `false` | When enabled, out-of-date pull request branches are updated before merging |
| `--repo` | - | Limit execution to specific repositories (may be repeated) |
| `--limit` | `0` | Maximum number of pull requests to merge in a single run (0 = unlimited) |
| `--json` | `false` | Output structured JSON instead of human-readable text |

## Authentication

Authentication is resolved in the following order:

1. `GITHUB_TOKEN` environment variable
2. GitHub CLI authentication via `gh auth token`

If neither is available, execution fails immediately with a clear error message.

### Required Permissions

The token must have the following permissions:

- Read repositories
- Read pull requests
- Read check runs and commit statuses
- Comment on pull requests (if rebasing is enabled)
- Merge pull requests (if not in dry-run mode)

## Dry Run Behavior

Dry run mode is enabled by default (`--dry-run=true`). In this mode:

- Full discovery and readiness evaluation is performed
- No mutations are made (no merges, no branch updates, no comments)
- Output clearly shows what actions **would** be taken

To perform actual merges, explicitly disable dry run:

```bash
ghprmerge --org myorg --source-branch dependabot/ --dry-run=false
```

## Rebase Behavior

When `--rebase` is enabled and a pull request branch is out of date:

- For Dependabot branches (prefixed with `dependabot/`): Posts a `@dependabot rebase` comment
- For other branches: Uses GitHub's update branch API to update the branch

After updating, the pull request must be re-evaluated before merging:
- Checks must pass again
- Branch must be confirmed up to date

## Pull Request Selection

Only pull requests meeting ALL of the following criteria are considered:

1. State is `open`
2. Base branch equals the repository's default branch
3. Head branch name contains the `--source-branch` pattern (substring matching)
4. Not marked as a draft

## Readiness Requirements

A pull request is considered ready to merge only if ALL of the following are true:

1. **Checks**: All check runs have a successful conclusion; all commit statuses are successful
2. **Mergeability**: No merge conflicts
3. **Branch Freshness**: Branch is fully up to date with the default branch

If any condition is not met, the pull request is skipped with a clear reason.

## Output

### Human-Readable Output (default)

```
ghprmerge - myorg
Source branch pattern: dependabot/
Mode: dry-run (no changes will be made)
Rebase: false

Repository: myorg/repo1 (default: main)
  PR #42: Bump lodash from 4.17.20 to 4.17.21
    Branch: dependabot/npm_and_yarn/lodash-4.17.21
    URL: https://github.com/myorg/repo1/pull/42
    Action: would merge
    Reason: all checks passing, branch up to date

Summary:
  Repositories scanned: 1
  Pull requests found: 1
  Would merge: 1
  Would rebase: 0
  Skipped: 0
```

### JSON Output

Use `--json` for structured output suitable for scripts and automation:

```json
{
  "metadata": {
    "org": "myorg",
    "source_branch": "dependabot/",
    "dry_run": true,
    "rebase": false
  },
  "repositories": [...],
  "summary": {
    "total_repositories": 1,
    "total_pull_requests": 1,
    "would_merge": 1,
    "skipped": 0
  }
}
```
