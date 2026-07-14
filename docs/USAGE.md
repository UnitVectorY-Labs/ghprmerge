---
layout: default
title: Usage
nav_order: 3
permalink: /usage
---

# Usage

## Command Line Reference

```
ghprmerge <command> --org <organization> [flags]
```

A subcommand is required. `--org` is required after that subcommand unless `GITHUB_ORG` is set. Choose `merge`, `rebase`, or `report`.

If you run `ghprmerge --help`, the CLI includes a short purpose line for each subcommand so you can quickly choose the right mode.

## Required Setup

| Flag | Default | Description |
|------|---------|-------------|
| `--org <organization>` | `GITHUB_ORG` env | GitHub organization to scan. Required unless `GITHUB_ORG` is set. |

## Filtering and Execution Controls

Use these to narrow the repositories or pull requests that a command considers.

| Flag | Default | Description |
|------|---------|-------------|
| `--repo <repository>` | - | Limit scanning to an exact repository name in the selected organization. Repeat for multiple repositories, such as `--repo api --repo web`. |
| `--author <login>` | `GHPRMERGE_AUTHOR` env | Include only PRs opened by this GitHub login, such as `app/dependabot`. |
| `--repo-limit <n>` | `0` | Process at most `n` repositories; `0` means unlimited. |

## Output Controls

These flags can be used with every subcommand.

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output structured JSON |
| `--no-color` | `false` | Disable colored output |
| `--no-progress` | `false` | Suppress progress bar output (useful for scripting, CI, and non-TTY environments) |
| `--version` | - | Show version information and exit |

## Commands

| Command | Description | Details |
|---------|-------------|---------|
| `merge` | Merge pull requests that are in a valid state | [MERGE.md](MERGE.md) |
| `rebase` | Update out-of-date branches | [REBASE.md](REBASE.md) |
| `report` | Scan and group open PRs by source branch | [REPORT.md](REPORT.md) |

## Command Behavior and Flags

### `merge`

Scans matching PRs and merges only those that are ready: not drafts, targeting the default branch, conflict-free, up to date (unless `--skip-rebase` is set), and passing checks.

| Flag | Description |
|------|-------------|
| `--source-branch <pattern>` | Required. Head-branch prefix to match; may be repeated. |
| `--skip-rebase` | Allow merge attempts when a branch is behind its default branch. |
| `--confirm` | Scan first, then prompt before merging candidates. |
| `--verbose` | Stream repository results during scanning, including repos with no matching pull requests. |

### `rebase`

Scans matching PRs and updates branches that are behind their repositories' default branch. It does not merge PRs.

| Flag | Description |
|------|-------------|
| `--source-branch <pattern>` | Required. Head-branch prefix to match; may be repeated. |
| `--confirm` | Scan first, then prompt before rebasing candidates. |
| `--verbose` | Stream repository results during scanning, including repos with no matching pull requests. |

### `report`

Read-only: scans open PRs, groups them by source branch, and reports matching groups. It never merges or rebases.

| Flag | Default | Description |
|------|---------|-------------|
| `--source-branch-prefix <prefixes>` | - | Comma-separated head-branch prefixes to include. |
| `--min-group-size <n>` | `2` | Include only groups with at least `n` PRs. |
| `--verbosity <level>` | `standard` | Text detail: `brief`, `standard`, or `verbose`. |

See each command's documentation for its full flag reference and examples.

## Help and Invalid Command Guidance

Use root help to discover commands and their purpose:

```bash
ghprmerge --help
```

If you provide an unknown subcommand, ghprmerge returns an error that includes the valid subcommands and what each one is for:

```bash
ghprmerge shipit
```

If you omit a subcommand and required command-specific flags, the error message includes subcommand guidance so you can recover quickly.

## Environment Variables

| Variable | Description |
|----------|-------------|
| `GITHUB_TOKEN` | GitHub personal access token (preferred) |
| `GITHUB_ORG` | Default organization (can be overridden by `--org`) |
| `GHPRMERGE_AUTHOR` | Default author filter (can be overridden by `--author`) |
| `GHPRMERGE_MIN_GROUP_SIZE` | Default minimum group size for the `report` command (can be overridden by `--min-group-size`) |

## Authentication

Authentication is resolved in order:

1. `GITHUB_TOKEN` environment variable
2. GitHub CLI via `gh auth token`

If neither is available, execution fails immediately.

### Required Permissions

- Read repositories
- Read pull requests
- Read check runs and commit statuses
- Comment on pull requests (for `rebase`)
- Merge pull requests (for `merge`)

## Sequential Processing

Repositories are processed **one at a time**. The tool:

- Never loads all org data before performing mutations
- Never operates on multiple repos in parallel
- Shows a progress bar as repositories are scanned
- When an action is performed (merge or rebase), the result is streamed to the console immediately, with the progress bar continuing below
- With `--verbose` (merge/rebase only), streams every repository result as soon as it is known
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
| `branch behind default` | Branch is out of date (in `merge` without `--skip-rebase`) |
| `branch updated, awaiting checks` | Rebase was done, waiting for checks |
| `insufficient permissions` | Token lacks required permissions |
| `API error` | GitHub API error (includes details) |

Pull requests with no checks configured are allowed to proceed. Pending checks still block merge decisions.

## Output Format

### Human-Readable (Default)

Output uses colored text and Unicode symbols for clear, scannable results:

- `✓` merged (green)
- `↻` rebased (yellow)
- `✗` failed (red)
- `⊘` skipped (dim)

A progress bar is shown during scanning:
```
  Scanning  15/25 [█████████████████████████████████                      ]  60%
```

Each action result is streamed to the console as soon as it completes, with the progress bar continuing below:
```
  ✓ myorg/repo1 #42 Bump lodash to 4.17.21
    merged ─ successfully merged (all checks passing)
  Scanning  15/25 [█████████████████████████████████                      ]  60%
```

With `--verbose` (merge and rebase only), repository results are emitted live during scanning, including repositories with no matching pull requests:
```
  ─ myorg/repo2 ─ no matching pull requests
  ✓ myorg/repo1 #42 Bump lodash to 4.17.21
    merged ─ all checks passing, branch up to date
```

A condensed summary line is printed at the end:
```
────────────────────────────────────────────────────
100 repos scanned │ 3 PRs found │ 1 merged │ 1 rebased │ 1 skipped
```

Use `--no-color` to disable ANSI color codes (useful for piping output or CI environments).

Use `--no-progress` to suppress the progress bar entirely. Final results and the summary line are still printed. This is recommended for scripting, CI pipelines, and any context where the output is captured or consumed by another program, since the carriage-return-based progress bar produces cluttered output in non-TTY environments.

### JSON Mode

```bash
ghprmerge merge --json --org myorg --source-branch dependabot/
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

## Version Information

```bash
ghprmerge --version
```

Displays the version of ghprmerge in the format:

```text
ghprmerge version vX.Y.Z (goX.Y, os/arch)
```

## Repo Limit Semantics

The `--repo-limit` flag limits the number of **repositories** processed:

```bash
ghprmerge merge --repo-limit 10 --org myorg --source-branch dependabot/
```

Output will show: `Limit: 10 repositories max`
