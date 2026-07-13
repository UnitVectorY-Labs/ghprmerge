---
layout: default
title: Examples
nav_order: 4
permalink: /examples
---

# Examples

## Show Version

```bash
ghprmerge --version
```

## Discover Commands and Their Purpose

Show top-level help with a brief summary of each subcommand:

```bash
ghprmerge --help
```

If you mistype a subcommand, ghprmerge will return a corrective error with valid subcommands:

```bash
ghprmerge shipit
# Error: unknown subcommand "shipit" (followed by valid subcommand guidance)
```

## Rebase Run

Update out-of-date branches without merging:

```bash
ghprmerge rebase --org myorg --source-branch dependabot/
```

For Dependabot branches, this posts a `@dependabot rebase` comment. For other branches, it uses GitHub's update branch API.

**Note**: `merge` and `rebase` are separate subcommands and cannot be combined. After rebasing, wait for checks to pass then run `merge`.

## Merge Run

Merge PRs that are already in a valid state (up-to-date, checks passing):

```bash
ghprmerge merge --org myorg --source-branch dependabot/
```

PRs that are behind the default branch will be skipped (use `rebase` first to update them, or `--skip-rebase` to merge anyway).

PRs with no checks configured are allowed to merge. PRs with pending checks are still skipped.

## Merge with Skip Rebase

Merge PRs even when they are behind the default branch:

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --skip-rebase
```

This is useful when your repository is configured to not require branches to be up-to-date before merging. The `--skip-rebase` flag allows merging without first updating the branch.

**Note**: `--skip-rebase` is only available under the `merge` subcommand. PRs with merge conflicts or failing checks will still be skipped.

## Confirmation Mode

Scan all repositories first, then prompt before taking actions:

```bash
ghprmerge rebase --org myorg --source-branch dependabot/ --confirm
```

Use `--confirm` with either `merge` or `rebase` to preview planned actions before execution. Pending actions are listed, and on confirmation, execution progress is shown with a progress bar.

```bash
ghprmerge merge --org myorg --source-branch dependabot/ --confirm
```

## Verbose Output

Stream repository decisions as each repository is scanned, including repositories with no matching pull requests:

```bash
ghprmerge merge --verbose --org myorg --source-branch dependabot/
```

By default, the scan stays quiet apart from the progress bar and only repositories with matching PRs are displayed after the scan completes.

## Verbose Confirmation Mode

Stream scan-time decisions, then clear them before showing the actions that were actually performed:

```bash
ghprmerge merge --verbose --org myorg --source-branch dependabot/ --confirm
```

This is useful when you want live visibility during the scan without leaving the terminal full of pending entries after you confirm.

## Disable Colored Output

Disable ANSI color codes for piping or CI:

```bash
ghprmerge merge --no-color --org myorg --source-branch dependabot/
```

## Suppress Progress Bar

Suppress the progress bar for scripting, CI pipelines, or when output is captured by another program:

```bash
ghprmerge merge --no-progress --org myorg --source-branch dependabot/
```

Final results and the summary line are still printed; only the carriage-return-based percentage lines are suppressed. Combine with `--no-color` for fully clean output in non-TTY environments:

```bash
ghprmerge merge --org myorg --no-color --no-progress --source-branch dependabot/
```

## Scoped Repository Run

Only process specific repositories:

```bash
ghprmerge merge --org myorg --repo repo1 --repo repo2 --source-branch dependabot/
```

## Multiple Source Branches

Match PRs from multiple source branch patterns in a single pass:

```bash
# Merge PRs matching either dependabot/ or repver/ patterns in a single pass
ghprmerge merge --org myorg --source-branch dependabot/ --source-branch repver/
```

Specifying multiple `--source-branch` flags scans all repositories once and matches PRs against each pattern. This reduces the number of scanning passes compared to running separate commands for each pattern. When multiple patterns match PRs in the same repository, the first matching pattern per repository takes priority for concurrent PR handling.

Multiple source branches also work with the `rebase` subcommand:

```bash
ghprmerge rebase --org myorg --source-branch dependabot/ --source-branch repver/
```

## Dependabot Focused Run

Match only Dependabot npm updates:

```bash
ghprmerge merge --org myorg --source-branch dependabot/npm_and_yarn/
```

Match only Go module updates:

```bash
ghprmerge merge --org myorg --source-branch dependabot/go_modules/
```

## Limited Run

Process at most 10 repositories:

```bash
ghprmerge merge --repo-limit 10 --org myorg --source-branch dependabot/
```

## JSON Output for Scripting

Get structured output for automation:

```bash
ghprmerge merge --json --org myorg --source-branch dependabot/ | jq '.summary'
```

Pipe to other tools:

```bash
ghprmerge merge --json --org myorg --source-branch dependabot/ | \
  jq -r '.repositories[].pull_requests[] | select(.action == "would merge") | .url'
```

## Report Mode

Scan open PRs across the organization and group them by source branch name:

```bash
ghprmerge report --org myorg
```

Filter to specific branch prefixes:

```bash
ghprmerge report --org myorg --source-branch-prefix dependabot/,repver/
```

Require at least 3 PRs in a group:

```bash
ghprmerge report --org myorg --min-group-size 3
```

The minimum group size can also be set via the `GHPRMERGE_MIN_GROUP_SIZE` environment variable:

```bash
export GHPRMERGE_MIN_GROUP_SIZE=3
ghprmerge report --org myorg
```

Get JSON output for scripting:

```bash
ghprmerge report --json --org myorg
```

Show only branch names and counts:

```bash
ghprmerge report --org myorg --verbosity brief
```

Include PR titles in the output:

```bash
ghprmerge report --org myorg --verbosity verbose
```

Scope the report to specific repositories:

```bash
ghprmerge report --org myorg --repo repo1 --repo repo2
```

## Using Environment Variables

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
export GITHUB_ORG=myorg

ghprmerge merge --source-branch dependabot/
```

## Filter by Author

Filter pull requests to only those opened by a specific author. This works with all subcommands.

Merge only PRs opened by the Dependabot app:

```bash
ghprmerge merge --author app/dependabot --org myorg --source-branch dependabot/
```

Rebase only PRs opened by a specific user:

```bash
ghprmerge rebase --author JaredHatfield --org myorg --source-branch feature/
```

Report on PRs from a specific author:

```bash
ghprmerge report --author app/dependabot --org myorg
```

The `--author` flag can also be set via the `GHPRMERGE_AUTHOR` environment variable:

```bash
export GHPRMERGE_AUTHOR=app/dependabot
ghprmerge merge --org myorg --source-branch dependabot/
```

## Complete Production Workflow

```bash
# Set authentication
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Step 1: Rebase out-of-date branches with confirmation
ghprmerge rebase --org myorg --source-branch dependabot/ --confirm

# Step 2: Wait for checks to complete (manual or scripted)
sleep 300

# Step 3: Merge ready PRs
ghprmerge merge --org myorg --source-branch dependabot/
```

## CI/CD Usage

Example GitHub Actions workflow:

```yaml
name: Auto-merge Dependabot
on:
  schedule:
    - cron: '0 9 * * 1'  # Every Monday at 9am
  workflow_dispatch:

jobs:
  merge:
    runs-on: ubuntu-latest
    steps:
      - name: Download ghprmerge
        run: |
          curl -L https://github.com/UnitVectorY-Labs/ghprmerge/releases/latest/download/ghprmerge_linux_amd64 -o ghprmerge
          chmod +x ghprmerge

      - name: Check version
        run: ./ghprmerge --version

      - name: Merge ready PRs
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
        run: ./ghprmerge merge --org myorg --no-progress --repo-limit 20 --source-branch dependabot/
```
