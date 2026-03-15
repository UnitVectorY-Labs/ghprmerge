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

### Shared Flags

These flags work in both normal mode and report mode.

| Flag | Default | Description |
|------|---------|-------------|
| `--org` | `GITHUB_ORG` env | GitHub organization to scan (required) |
| `--repo` | - | Limit to specific repositories (repeatable) |
| `--repo-limit` | `0` | Maximum repositories to process (0 = unlimited) |
| `--json` | `false` | Output structured JSON |
| `--verbose` | `false` | Stream repository results during scanning, including repos with no matching pull requests |
| `--no-color` | `false` | Disable colored output |
| `--version` | - | Show version information and exit |

### Normal Mode Flags

These flags are only valid in normal mode (without `--report`).

| Flag | Default | Mutates State | Description |
|------|---------|---------------|-------------|
| `--source-branch` | - | No | Branch pattern to match PR head branches (required in normal mode) |
| `--rebase` | `false` | **Yes** | Update out-of-date branches (mutually exclusive with --merge and --skip-rebase) |
| `--merge` | `false` | **Yes** | Merge PRs that are in a valid state (mutually exclusive with --rebase) |
| `--skip-rebase` | `false` | **Yes** | Skip rebase check and merge PRs that are behind (requires --merge, mutually exclusive with --rebase) |
| `--confirm` | `false` | No | Scan all repos first, then prompt for confirmation |

### Report Mode Flags

These flags are only valid when `--report` is used.

| Flag | Default | Description |
|------|---------|-------------|
| `--report` | `false` | Report mode: scan open PRs and group by source branch name |
| `--source-branch-prefix` | - | Comma-separated list of branch prefixes to include in report |
| `--min-group-size` | `2` | Minimum number of PRs in a group to include in report |
| `--verbosity` | `standard` | Report output verbosity: `brief`, `standard`, or `verbose` |

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

- Scans all repositories first with a progress bar (no actions taken during scan)
- In the default view, displays a list of pending actions before prompting
- With `--verbose`, streams scan-time repository results as they are discovered
- Prompts for user confirmation before executing
- On confirmation, clears the pending scan/prompt output and streams each action result as it completes, with the progress bar continuing below
- If no actions are pending, prints the matching repository results and skip reasons instead of prompting
- Useful for reviewing changes before taking action

### Report Mode

```bash
ghprmerge --org myorg --report
```

- Scans all repos using the same discovery logic as normal mode
- Lists all open PRs (non-draft, targeting default branch)
- Groups PRs by exact source branch name
- Filters by prefix if `--source-branch-prefix` is set
- Filters out groups smaller than `--min-group-size` (default: 2)
- Sorts by descending count, then ascending branch name for ties
- Evaluates each PR's status using the same logic as normal mode (check status, branch status)
- **No mutations performed**

**Flag restrictions**: Normal mode flags (`--source-branch`, `--rebase`, `--merge`, `--skip-rebase`, `--confirm`) cannot be used with `--report`. Report mode flags (`--source-branch-prefix`, `--min-group-size`, `--verbosity`) can only be used with `--report`.

#### Text Output Verbosity

The `--verbosity` flag controls the level of detail in text output:

- **`brief`**: Branch name and PR count only
- **`standard`** (default): Branch name, count, and for each PR: repo name, PR number, status
- **`verbose`**: Standard output plus PR title

#### JSON Output

With `--json`, report mode outputs:

```json
{
  "groups": [
    {
      "sourceBranch": "dependabot/go_modules/foo-1.2.3",
      "count": 3,
      "pullRequests": [
        {
          "repository": "repo-a",
          "number": 123,
          "status": "passing",
          "title": "Bump foo",
          "url": "https://github.com/..."
        }
      ]
    }
  ]
}
```

#### Status Values

Status is the same assessment logic as normal ghprmerge: `passing`, `needs-rebase`, `conflict`, `checks failing`, `checks pending`, `no checks configured`, `error`.

#### Empty Results

When no grouped source branches are found, text output shows "No grouped source branches found.", JSON outputs an empty `groups` array, and exit code is 0.

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
- When an action is performed (merge or rebase), the result is streamed to the console immediately as it happens, with the progress bar continuing below
- In analysis mode (no `--merge` or `--rebase`), prints matching repository results after the scan completes
- With `--verbose`, streams every repository result as soon as it is known
- With `--confirm`, streams action results during the execution phase after the user confirms

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

In analysis mode (no `--merge` or `--rebase`), after scanning only repositories with matching PRs are shown. Each matching PR is displayed with its action and details:
```
  ✓ myorg/repo1 #42 Bump lodash to 4.17.21
    would merge ─ all checks passing, branch up to date
```

When actions are performed (`--merge` or `--rebase`), each result is streamed to the console as soon as the action completes, with the progress bar continuing below:
```
  ✓ myorg/repo1 #42 Bump lodash to 4.17.21
    merged ─ successfully merged (all checks passing)
Scanning [██████████████████░░░░░░░░░░░░] 15/25 (60%)
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
