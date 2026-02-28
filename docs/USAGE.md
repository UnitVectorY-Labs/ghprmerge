---
layout: default
title: Usage
nav_order: 3
permalink: /usage
---

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
| `--rebase` | `false` | **Yes** | Update out-of-date branches (mutually exclusive with --merge and --skip-rebase) |
| `--merge` | `false` | **Yes** | Merge PRs that are in a valid state (mutually exclusive with --rebase) |
| `--skip-rebase` | `false` | **Yes** | Skip rebase check and merge PRs that are behind (requires --merge, mutually exclusive with --rebase) |
| `--repo` | - | No | Limit to specific repositories (repeatable) |
| `--repo-limit` | `0` | No | Maximum repositories to process (0 = unlimited) |
| `--json` | `false` | No | Output structured JSON |
| `--confirm` | `false` | No | Scan all repos first, then prompt for confirmation |
| `--verbose` | `false` | No | Stream repository results during scanning, including repos with no matching pull requests |
| `--no-color` | `false` | No | Disable colored output |
| `--version` | - | No | Show version information and exit |

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
- **Does NOT merge** - rebase and merge are mutually exclusive
- After rebasing, run a separate `--merge` command once checks pass

### Merge Only

```bash
ghprmerge --org myorg --source-branch dependabot/ --merge
```

- Merges PRs that are already in a valid state (up-to-date, checks passing)
- **Does NOT attempt any rebases** - rebase and merge are mutually exclusive
- Skips PRs that are behind with a clear reason

### Merge with Skip Rebase

```bash
ghprmerge --org myorg --source-branch dependabot/ --merge --skip-rebase
```

- Merges PRs even if they are behind the default branch
- Still requires all checks to be passing and no merge conflicts
- Useful when the repository doesn't require branches to be up-to-date before merging
- **Does NOT rebase** - the `--skip-rebase` flag skips the rebase requirement entirely
- Cannot be used with `--rebase` (mutually exclusive)

### Confirmation Mode

```bash
ghprmerge --org myorg --source-branch dependabot/ --rebase --confirm
```

- Scans all repositories first with a progress bar
- In the default view, displays a list of pending actions before prompting
- With `--verbose`, streams scan-time repository results as they are discovered
- Prompts for user confirmation before executing
- On confirmation, clears the pending scan/prompt output and shows execution progress with actual results
- If no actions are pending, prints the matching repository results and skip reasons instead of prompting
- Useful for reviewing changes before taking action

### Verbose Output

```bash
ghprmerge --org myorg --source-branch dependabot/ --verbose
```

- Streams repository results live while the scan runs
- Includes repositories with no matching pull requests
- By default, the scan stays quiet apart from the progress bar, and only repositories with matching PRs are shown after the scan

### Disable Colors

```bash
ghprmerge --org myorg --source-branch dependabot/ --no-color
```

- Disables ANSI color codes in terminal output
- Useful for piping output to files or running in CI environments

## Version Information

```bash
ghprmerge --version
```

Displays the version of ghprmerge.

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
- Shows a progress bar as repositories are scanned
- In default output, prints matching repository results after the scan completes
- With `--verbose`, streams every repository result as soon as it is known

## Archived Repository Handling

Archived repositories are automatically excluded during repository discovery and are never processed. Since archived repositories cannot be modified, they are filtered out during discovery.

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
| `branch behind default` | Branch is out of date (and `--rebase` not set) |
| `branch updated, awaiting checks` | Rebase was done, waiting for checks |
| `insufficient permissions` | Token lacks required permissions |
| `API error` | GitHub API error (includes details) |

Pull requests with no checks configured are allowed to proceed. Pending checks still block merge decisions.

## Output Format

### Human-Readable (Default)

Output uses colored text and Unicode symbols for clear, scannable results:

- `✓` merged / would merge (green)
- `↻` rebased / would rebase (yellow)
- `✗` failed (red)
- `⊘` skipped (dim)

A progress bar is shown during scanning:
```
Scanning [██████████████████░░░░░░░░░░░░] 15/25 (60%)
```

By default, after scanning only repositories with matching PRs are shown. Each matching PR is displayed with its action and details:
```
  ✓ myorg/repo1 #42 Bump lodash to 4.17.21
    would merge ─ all checks passing, branch up to date
```

With `--verbose`, repository results are emitted live during scanning, including repositories with no matching pull requests:
```
  ─ myorg/repo2 ─ no matching pull requests
  ✓ myorg/repo1 #42 Bump lodash to 4.17.21
    would merge ─ all checks passing, branch up to date
```

A condensed summary line is printed at the end:
```
────────────────────────────────────────────────────
100 repos scanned │ 3 PRs found │ 1 merged │ 1 rebased │ 1 skipped
```

Use `--no-color` to disable ANSI color codes (useful for piping output or CI environments).

### JSON Mode

```bash
ghprmerge --org myorg --source-branch dependabot/ --json
```

Outputs structured JSON with:
- Run metadata (org, mode, limits)
- Per-repository results
- Per-PR decisions with action and reason
- Summary statistics grouped by skip reason

## Dependabot Branch Handling

For branches with the `dependabot/` prefix:
- **Rebase method**: Posts `@dependabot rebase` comment instead of directly updating the branch
- Dependabot will then perform the rebase on its own

For non-Dependabot branches:
- **Rebase method**: Uses GitHub's update branch API directly
