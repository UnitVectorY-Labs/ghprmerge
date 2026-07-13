---
layout: default
title: Merge Command
nav_order: 5
permalink: /merge
---

# Merge Command

The `merge` subcommand merges pull requests that are in a valid state across repositories in a GitHub organization.

## Synopsis

```
ghprmerge merge --org <organization> [flags]
```

The merge subcommand scans repositories for PRs matching the specified source branch patterns and merges those that are up-to-date, have passing checks, and have no merge conflicts. It does **not** rebase branches — merge and rebase are separate subcommands.

## Required Setup

| Flag | Default | Description |
|------|---------|-------------|
| `--org <organization>` | `GITHUB_ORG` env | GitHub organization to scan. Required unless `GITHUB_ORG` is set. |

## Filtering and Execution Controls

| Flag | Default | Description |
|------|---------|-------------|
| `--repo <repository>` | - | Limit scanning to an exact repository name in the organization; may be repeated. |
| `--author <login>` | `GHPRMERGE_AUTHOR` env | Include only PRs opened by this GitHub login. |
| `--repo-limit <n>` | `0` | Process at most `n` repositories; `0` means unlimited. |

## Output Controls

All output flags can be used with `merge`.

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output structured JSON instead of human-readable text. |
| `--verbose` | `false` | Show repositories with no matching PRs as they are scanned. |
| `--no-color` | `false` | Disable ANSI color output. |
| `--no-progress` | `false` | Suppress progress-bar output for CI or scripts. |

## Merge Flags

These flags are placed after `merge`.

| Flag | Default | Description |
|------|---------|-------------|
| `--source-branch` | - | Branch name pattern to match PR head branches (required, repeatable) |
| `--skip-rebase` | `false` | Skip rebase check and merge PRs that are behind the default branch |
| `--confirm` | `false` | Scan all repos first, then prompt for confirmation before merging |

## Behavior

The merge subcommand processes repositories sequentially and evaluates each matching PR against the following criteria:

- **Up-to-date**: The PR branch must not be behind the default branch. PRs that are behind are skipped unless `--skip-rebase` is used.
- **Checks passing**: All required status checks must have completed successfully. PRs with failing checks are skipped.
- **No merge conflicts**: PRs with merge conflicts are skipped.
- **Not a draft**: Draft PRs are excluded.
- **Targets default branch**: Only PRs targeting the repository's default branch are considered.

PRs with **no checks configured** are allowed to merge. PRs with **pending checks** are skipped — the tool will not wait for checks to finish.

The merge subcommand does **not** rebase branches. If a PR is behind the default branch, use the `rebase` subcommand first to bring it up-to-date, then run `merge` after checks pass. Merge and rebase are intentionally separate operations to provide explicit control over each step.

## Skip Rebase

By default, the merge subcommand skips PRs whose branches are behind the default branch. The `--skip-rebase` flag overrides this behavior and allows merging PRs even when they are not up-to-date.

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --skip-rebase
```

This is useful when:

- The repository does not require branches to be up-to-date before merging
- You trust that the PR's changes are compatible with the current default branch
- You want to merge quickly without a rebase cycle

**Restrictions**: `--skip-rebase` cannot be used with the `rebase` subcommand. PRs with merge conflicts or failing checks are still skipped regardless of this flag.

## Confirmation Mode

The `--confirm` flag changes the execution flow to a two-phase process:

1. **Scan phase**: All repositories are scanned and candidate PRs are identified. No mutations are performed during this phase. A progress bar is displayed during scanning.
2. **Prompt phase**: A summary of pending merge actions is displayed, and the user is prompted for confirmation before proceeding.

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --confirm
```

On confirmation:

- The pending scan output is cleared
- Each merge action result is streamed to the console as it completes
- The progress bar continues below the streamed results

If no actions are pending (e.g., all PRs are already merged or skipped), the matching repository results and skip reasons are printed instead of prompting.

With `--verbose`, scan-time repository results are streamed live as they are discovered, giving visibility into the scan before the prompt appears.

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --confirm --verbose
```

## Multiple Source Branches

The `--source-branch` flag can be specified multiple times to match PRs across different branch name patterns in a single run:

```bash
ghprmerge merge --org myorg --source-branch dependabot/npm_and_yarn/ --source-branch dependabot/go_modules/
```

### Matching behavior

- Multiple `--source-branch` patterns are matched during a single scan of each repository, reducing the number of passes required.
- If multiple source branch patterns match PRs in the **same repository**, only the first matching pattern (by the order specified on the command line) is used. Subsequent matches in that repository are skipped.
- This prevents concurrent modification issues where merging one PR could invalidate the branch state of another PR in the same repository.

### Pattern ordering

The order of `--source-branch` flags matters. Place higher-priority patterns first:

```bash
# Prioritize Go module updates over npm updates
ghprmerge merge --org myorg \
  --source-branch dependabot/go_modules/ \
  --source-branch dependabot/npm_and_yarn/
```

In this example, if a repository has both a Go module update PR and an npm update PR, only the Go module PR will be considered for merging.

## Examples

### Basic merge

Merge all ready Dependabot PRs across the organization:

```bash
ghprmerge merge --org myorg --source-branch dependabot/
```

### Merge by author

Merge only PRs opened by `app/dependabot`:

```bash
ghprmerge merge --author app/dependabot --org myorg --source-branch dependabot/
```

Merge only PRs opened by a specific user:

```bash
ghprmerge merge --author JaredHatfield --org myorg --source-branch feature/
```

### Merge specific repos

Limit merging to specific repositories:

```bash
ghprmerge merge --org myorg --repo repo1 --repo repo2 --source-branch dependabot/
```

### Merge with skip rebase

Merge PRs even when they are behind the default branch:

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --skip-rebase
```

### Merge with confirmation

Review planned actions before executing:

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --confirm
```

### Merge with verbose confirmation

Stream scan results live, then confirm before merging:

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --confirm --verbose
```

### Merge multiple branch patterns

Merge PRs from multiple Dependabot ecosystems:

```bash
ghprmerge merge --org myorg \
  --source-branch dependabot/npm_and_yarn/ \
  --source-branch dependabot/go_modules/ \
  --source-branch dependabot/pip/
```

### Merge with repo limit

Process at most 10 repositories:

```bash
ghprmerge merge --repo-limit 10 --org myorg --source-branch dependabot/
```

### JSON output for scripting

Get structured output for automation pipelines:

```bash
ghprmerge merge --json --org myorg --source-branch dependabot/ | jq '.summary'
```

### Disable colored output

Useful for CI environments or piping to a file:

```bash
ghprmerge merge --no-color --org myorg --source-branch dependabot/
```

### Production workflow

A typical two-step workflow using rebase and merge as separate operations:

```bash
# Step 1: Rebase out-of-date branches
ghprmerge rebase --org myorg --source-branch dependabot/ --confirm

# Step 2: Wait for checks to pass
sleep 300

# Step 3: Merge ready PRs
ghprmerge merge --org myorg --source-branch dependabot/
```
