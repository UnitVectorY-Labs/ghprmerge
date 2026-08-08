---
layout: default
title: Close Command
nav_order: 7
permalink: /close
---

# Close Command

The `close` subcommand closes matching pull requests across repositories in a GitHub organization. It can optionally delete the PR source branch after the close succeeds.

## Synopsis

```
ghprmerge close --org <organization> --source-branch <pattern> [flags]
```

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

All output flags can be used with `close`.

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | `false` | Output structured JSON instead of human-readable text. |
| `--verbose` | `false` | Show repositories with no matching PRs as they are scanned. |
| `--no-color` | `false` | Disable ANSI color output. |
| `--no-progress` | `false` | Suppress progress-bar output for CI or scripts. |

## Close Flags

These flags are placed after `close`.

| Flag | Default | Description |
|------|---------|-------------|
| `--source-branch <pattern>` | - | Branch name pattern to match PR head branches (required, repeatable). |
| `--delete-source-branch` | `false` | After a successful close, delete the source branch from the PR's head repository. |
| `--confirm` | `false` | Scan all repositories first, then prompt for confirmation before closing. |

## Behavior

The close subcommand processes repositories sequentially and closes matching PRs. A PR is eligible when it is open, is not a draft, and targets its repository's default branch.

Unlike `merge`, close eligibility does not depend on check status, merge conflicts, or whether the source branch is up to date. The command closes the PR without merging it.

## Delete Source Branches

Use `--delete-source-branch` when the source branch should be removed after a PR has been closed:

```bash
ghprmerge close --org myorg --source-branch stale/ --delete-source-branch
```

Branch deletion occurs only after the close succeeds. The command deletes the branch from the PR's head repository, including a fork when applicable. If closing the PR fails, its source branch is not deleted. If deletion fails, the PR remains closed and the result reports the deletion as a partial failure.

## Confirmation Mode

The `--confirm` flag changes execution to a two-phase process:

1. **Scan phase**: The command finds eligible PRs but makes no changes.
2. **Prompt phase**: It displays pending close actions and asks for confirmation.

```bash
ghprmerge close --org myorg --source-branch stale/ --delete-source-branch --confirm
```

After confirmation, each close result is streamed immediately. When `--delete-source-branch` is set, the source-branch deletion is attempted only after the corresponding close succeeds.

## Multiple Source Branches

Repeat `--source-branch` to match multiple branch patterns in one scan:

```bash
ghprmerge close --org myorg \
  --source-branch stale/ \
  --source-branch feature/abandoned/
```

If multiple patterns match PRs in the same repository, the first matching pattern specified on the command line takes priority.

## Examples

### Close by author

Close only Dependabot PRs opened by the Dependabot app:

```bash
ghprmerge close --author 'dependabot[bot]' --org myorg --source-branch dependabot/
```

### Close in selected repositories

```bash
ghprmerge close --org myorg --repo api --repo web --source-branch stale/
```

### Close with confirmation

```bash
ghprmerge close --org myorg --source-branch stale/ --confirm
```

### JSON output

```bash
ghprmerge close --json --org myorg --source-branch stale/ | jq '.summary'
```
