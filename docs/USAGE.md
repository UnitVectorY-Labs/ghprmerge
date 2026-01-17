# Usage

## Command Line Reference

```
ghprmerge [flags]
```

## Flags Reference

| Flag | Default | Mutates State | Description |
|------|---------|---------------|-------------|
| `--org` | `GITHUB_ORG` env | No | GitHub organization to scan (required) |
| `--source-branch` | - | No | Branch pattern to match PR head branches (required) |
| `--rebase` | `false` | **Yes** | Update out-of-date branches |
| `--merge` | `false` | **Yes** | Merge PRs that are in a valid state |
| `--repo` | - | No | Limit to specific repositories (repeatable) |
| `--repo-limit` | `0` | No | Maximum repositories to process (0 = unlimited) |
| `--json` | `false` | No | Output structured JSON |

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token (preferred) |
| `GITHUB_ORG` | Default organization (can be overridden by `--org`) |

## Flag Combinations

### Default (Analysis Only)

```bash
ghprmerge --org myorg --source-branch dependabot/
```

- Scans repositories sequentially
- Evaluates candidate PRs
- Reports what would be rebased and merged
- **No mutations performed**

### Rebase Only

```bash
ghprmerge --org myorg --source-branch dependabot/ --rebase
```

- Updates out-of-date branches
- For Dependabot branches: posts `@dependabot rebase` comment
- For other branches: uses GitHub's update branch API
- **Does NOT merge** (use `--merge` to also merge)
- Reports which PRs are merge-ready after update

### Merge Only

```bash
ghprmerge --org myorg --source-branch dependabot/ --merge
```

- Merges PRs that are already in a valid state
- **Does NOT attempt any rebases**
- Skips PRs that are behind with a clear reason

### Rebase and Merge

```bash
ghprmerge --org myorg --source-branch dependabot/ --rebase --merge
```

- Updates branches where needed
- Merges PRs that are valid after evaluation
- If checks become pending after rebase, PR is reported as "updated, awaiting checks" and skipped for merging

## Repo Limit Semantics

The `--repo-limit` flag limits the number of **repositories** processed:

```bash
ghprmerge --org myorg --source-branch dependabot/ --repo-limit 10
```

Output will show: `Limit: 10 repositories max`

## Authentication

Authentication is resolved in order:

1. `GITHUB_TOKEN` environment variable
2. GitHub CLI via `gh auth token`

If neither is available, execution fails immediately.

### Required Permissions

- Read repositories
- Read pull requests
- Read check runs and commit statuses
- Comment on pull requests (if `--rebase` is used)
- Merge pull requests (if `--merge` is used)

## Sequential Processing

Repositories are processed **one at a time**. The tool:

- Never loads all org data before performing mutations
- Never operates on multiple repos in parallel
- Prints results for each repo immediately after processing

## Skip Reasons

When a PR is skipped, one of these reasons is shown:

| Reason | Description |
|--------|-------------|
| `not targeting default branch` | PR base is not the repo's default branch |
| `branch does not match source pattern` | Head branch doesn't match `--source-branch` |
| `draft PR` | PR is marked as draft |
| `merge conflict` | PR has merge conflicts |
| `checks failing` | One or more checks failed (includes check name) |
| `checks pending` | Checks are still running |
| `no checks found` | No checks configured for this PR |
| `branch behind default` | Branch is out of date (and `--rebase` not set) |
| `branch updated, awaiting checks` | Rebase was done, waiting for checks |
| `insufficient permissions` | Token lacks required permissions |
| `API error` | GitHub API error (includes details) |

## Output Format

### Human-Readable (Default)

Clear sections per repository with consistent status symbols:

- `✓` merged / would merge
- `↻` rebased / would rebase  
- `✗` failed
- `⊘` skipped

### JSON Mode

```bash
ghprmerge --org myorg --source-branch dependabot/ --json
```

Outputs structured JSON with:
- Run metadata (org, mode, limits)
- Per-repository results
- Per-PR decisions with action and reason
- Summary statistics grouped by skip reason
