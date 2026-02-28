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

## Default Analysis Run

Preview what would happen without making any changes:

```bash
ghprmerge --org myorg --source-branch dependabot/
```

This scans all repositories, evaluates PRs, and reports what would be rebased and merged.

## Rebase Only Run

Update out-of-date branches without merging:

```bash
ghprmerge --org myorg --source-branch dependabot/ --rebase
```

For Dependabot branches, this posts a `@dependabot rebase` comment. For other branches, it uses GitHub's update branch API.

**Note**: `--rebase` and `--merge` are mutually exclusive. After rebasing, wait for checks to pass then run with `--merge`.

## Merge Only Run

Merge PRs that are already in a valid state (up-to-date, checks passing):

```bash
ghprmerge --org myorg --source-branch dependabot/ --merge
```

PRs that are behind the default branch will be skipped (use `--rebase` first to update them, or `--skip-rebase` to merge anyway).

## Merge with Skip Rebase

Merge PRs even when they are behind the default branch:

```bash
ghprmerge --org myorg --source-branch dependabot/ --merge --skip-rebase
```

This is useful when your repository is configured to not require branches to be up-to-date before merging. The `--skip-rebase` flag allows merging without first updating the branch.

**Note**: `--skip-rebase` requires `--merge` and cannot be used with `--rebase`. PRs with merge conflicts or failing checks will still be skipped.

## Confirmation Mode

Scan all repositories first, then prompt before taking actions:

```bash
ghprmerge --org myorg --source-branch dependabot/ --rebase --confirm
```

This is useful when you want to review the planned actions before execution. Pending actions are listed, and on confirmation, execution progress is shown with a progress bar.

## Verbose Output

Show all repositories, including those with no matching pull requests:

```bash
ghprmerge --org myorg --source-branch dependabot/ --verbose
```

By default, only repositories with matching PRs are displayed in the output.

## Disable Colored Output

Disable ANSI color codes for piping or CI:

```bash
ghprmerge --org myorg --source-branch dependabot/ --no-color
```

## Scoped Repository Run

Only process specific repositories:

```bash
ghprmerge --org myorg --source-branch dependabot/ --repo repo1 --repo repo2 --merge
```

## Dependabot Focused Run

Match only Dependabot npm updates:

```bash
ghprmerge --org myorg --source-branch dependabot/npm_and_yarn/ --merge
```

Match only Go module updates:

```bash
ghprmerge --org myorg --source-branch dependabot/go_modules/ --merge
```

## Limited Run

Process at most 10 repositories:

```bash
ghprmerge --org myorg --source-branch dependabot/ --repo-limit 10 --merge
```

## JSON Output for Scripting

Get structured output for automation:

```bash
ghprmerge --org myorg --source-branch dependabot/ --json | jq '.summary'
```

Pipe to other tools:

```bash
ghprmerge --org myorg --source-branch dependabot/ --json | \
  jq -r '.repositories[].pull_requests[] | select(.action == "would merge") | .url'
```

## Using Environment Variables

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
export GITHUB_ORG=myorg

ghprmerge --source-branch dependabot/
```

## Complete Production Workflow

```bash
# Set authentication
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Step 1: Analyze what's available
ghprmerge --org myorg --source-branch dependabot/

# Step 2: Rebase out-of-date branches with confirmation
ghprmerge --org myorg --source-branch dependabot/ --rebase --confirm

# Step 3: Wait for checks to complete (manual or scripted)
sleep 300

# Step 4: Merge ready PRs
ghprmerge --org myorg --source-branch dependabot/ --merge
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

      - name: Analyze
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
        run: ./ghprmerge --org myorg --source-branch dependabot/ --json

      - name: Merge ready PRs
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
        run: ./ghprmerge --org myorg --source-branch dependabot/ --merge --repo-limit 20
```
