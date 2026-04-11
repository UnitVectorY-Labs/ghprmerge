---
layout: default
title: Rebase Command
nav_order: 6
permalink: /rebase
---

# Rebase Command

The `rebase` subcommand updates pull request branches that are behind the default branch across repositories in a GitHub organization.

## Synopsis

```
ghprmerge [global-flags] rebase [rebase-flags]
```

The rebase subcommand scans repositories for PRs matching the specified source branch patterns and brings out-of-date branches up-to-date with the default branch. It does **not** merge PRs — rebase and merge are separate subcommands.

## Flags

### Global Flags

These flags are placed before the `rebase` subcommand.

| Flag | Default | Description |
|------|---------|-------------|
| `--org` | `GITHUB_ORG` env | GitHub organization to scan (required) |
| `--repo` | - | Limit to specific repositories (repeatable) |
| `--repo-limit` | `0` | Maximum repositories to process (0 = unlimited) |
| `--json` | `false` | Output structured JSON |
| `--verbose` | `false` | Show all repos including those with no matching PRs |
| `--no-color` | `false` | Disable colored output |

### Rebase Flags

These flags are placed after the `rebase` subcommand.

| Flag | Default | Description |
|------|---------|-------------|
| `--source-branch` | - | Branch name pattern to match PR head branches (required, repeatable) |
| `--confirm` | `false` | Scan all repos first, then prompt for confirmation before rebasing |
| `--repo` | - | Additional repo filter, in addition to the global `--repo` (repeatable) |

## Behavior

The rebase subcommand processes repositories sequentially and updates branches that are behind the default branch. Unlike the merge subcommand, failing or pending checks are **not** blocking — rebasing may resolve the issues causing check failures, so the tool does not gate on check status.

Key behavioral properties:

- **Out-of-date branches only**: Branches that are already up-to-date with the default branch are skipped.
- **Not a draft**: Draft PRs are excluded.
- **Targets default branch**: Only PRs targeting the repository's default branch are considered.
- **No merging**: The rebase subcommand never merges a PR. Rebase and merge are intentionally separate operations to provide explicit control over each step.
- **Checks are not blocking**: Since rebasing can resolve failing checks (e.g., a conflict with a recently merged change), the rebase subcommand does not skip PRs based on check status. This differs from the merge subcommand, which requires all checks to pass.

After rebasing, run the `merge` subcommand once checks have passed:

```bash
# Step 1: Rebase out-of-date branches
ghprmerge --org myorg rebase --source-branch dependabot/

# Step 2: Wait for checks to pass
sleep 300

# Step 3: Merge ready PRs
ghprmerge --org myorg merge --source-branch dependabot/
```

## Dependabot Handling

The rebase subcommand uses different strategies depending on whether the branch is managed by Dependabot:

### Dependabot branches (prefix `dependabot/`)

For branches with the `dependabot/` prefix, the tool posts a comment containing `@dependabot rebase` on the pull request. Dependabot detects this comment and performs the rebase on its own. This is the correct approach because Dependabot manages its own branches and direct updates via the API would conflict with its workflow.

```bash
ghprmerge --org myorg rebase --source-branch dependabot/
```

### Non-Dependabot branches

For branches that do not have the `dependabot/` prefix, the tool uses GitHub's update branch API directly to bring the branch up-to-date with the default branch.

```bash
ghprmerge --org myorg rebase --source-branch feature/batch-update
```

## Confirmation Mode

The `--confirm` flag changes the execution flow to a two-phase process:

1. **Scan phase**: All repositories are scanned and candidate PRs are identified. No mutations are performed during this phase. A progress bar is displayed during scanning.
2. **Prompt phase**: A summary of pending rebase actions is displayed, and the user is prompted for confirmation before proceeding.

```bash
ghprmerge --org myorg rebase --source-branch dependabot/ --confirm
```

On confirmation:

- The pending scan output is cleared
- Each rebase action result is streamed to the console as it completes
- The progress bar continues below the streamed results

If no actions are pending (e.g., all PRs are already up-to-date or skipped), the matching repository results and skip reasons are printed instead of prompting.

With `--verbose`, scan-time repository results are streamed live as they are discovered, giving visibility into the scan before the prompt appears.

```bash
ghprmerge --org myorg rebase --source-branch dependabot/ --confirm --verbose
```

## Multiple Source Branches

The `--source-branch` flag can be specified multiple times to match PRs across different branch name patterns in a single run:

```bash
ghprmerge --org myorg rebase --source-branch dependabot/npm_and_yarn/ --source-branch dependabot/go_modules/
```

### Matching behavior

- Multiple `--source-branch` patterns are matched during a single scan of each repository, reducing the number of passes required.
- If multiple source branch patterns match PRs in the **same repository**, only the first matching pattern (by the order specified on the command line) is used. Subsequent matches in that repository are skipped.
- This prevents concurrent modification issues where rebasing one PR could invalidate the branch state of another PR in the same repository.

### Pattern ordering

The order of `--source-branch` flags matters. Place higher-priority patterns first:

```bash
# Prioritize Go module updates over npm updates
ghprmerge --org myorg rebase \
  --source-branch dependabot/go_modules/ \
  --source-branch dependabot/npm_and_yarn/
```

In this example, if a repository has both a Go module update PR and an npm update PR, only the Go module PR will be considered for rebasing.

## Examples

### Basic rebase

Rebase all out-of-date Dependabot PRs across the organization:

```bash
ghprmerge --org myorg rebase --source-branch dependabot/
```

### Rebase specific repos

Limit rebasing to specific repositories:

```bash
ghprmerge --org myorg --repo repo1 --repo repo2 rebase --source-branch dependabot/
```

### Rebase with confirmation

Review planned actions before executing:

```bash
ghprmerge --org myorg rebase --source-branch dependabot/ --confirm
```

### Rebase with verbose confirmation

Stream scan results live, then confirm before rebasing:

```bash
ghprmerge --org myorg rebase --source-branch dependabot/ --confirm --verbose
```

### Rebase multiple branch patterns

Rebase PRs from multiple Dependabot ecosystems:

```bash
ghprmerge --org myorg rebase \
  --source-branch dependabot/npm_and_yarn/ \
  --source-branch dependabot/go_modules/ \
  --source-branch dependabot/pip/
```

### Rebase with repo limit

Process at most 10 repositories:

```bash
ghprmerge --org myorg --repo-limit 10 rebase --source-branch dependabot/
```

### Rebase non-Dependabot branches

Update a custom branch pattern using the GitHub update branch API directly:

```bash
ghprmerge --org myorg rebase --source-branch feature/batch-update
```

### JSON output for scripting

Get structured output for automation pipelines:

```bash
ghprmerge --org myorg --json rebase --source-branch dependabot/ | jq '.summary'
```

### Disable colored output

Useful for CI environments or piping to a file:

```bash
ghprmerge --org myorg --no-color rebase --source-branch dependabot/
```

### Production workflow

A typical two-step workflow using rebase and merge as separate operations:

```bash
# Step 1: Rebase out-of-date branches
ghprmerge --org myorg rebase --source-branch dependabot/ --confirm

# Step 2: Wait for checks to pass
sleep 300

# Step 3: Merge ready PRs
ghprmerge --org myorg merge --source-branch dependabot/
```
