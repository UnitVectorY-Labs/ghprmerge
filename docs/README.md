---
layout: default
title: ghprmerge
nav_order: 1
permalink: /
---

# ghprmerge

## Purpose

ghprmerge solves the problem of merging many similar pull requests across a GitHub organization. When you have dozens or hundreds of repositories with Dependabot (or similar automated) PRs, manually reviewing and merging each one becomes impractical.

The tool provides three subcommands:

- **`merge`** — merge ready pull requests across an organization
- **`rebase`** — update out-of-date PR branches across an organization
- **`report`** — scan open PRs and group them by source branch name, helping you identify common updates that span multiple repositories

## Safety Model

ghprmerge is designed to be **safe by default**:

1. **Explicit subcommands** - Use `merge` to merge PRs, `rebase` to update branches, or `report` for a read-only overview
2. **Confirmation mode** - Use `--confirm` with `merge` or `rebase` to preview what would happen before executing
3. **Strict readiness checks** - A PR is only considered ready if:
   - All check runs have a successful conclusion (including non-required checks), or no checks are configured at all
   - All commit status contexts are successful, or no statuses are configured at all
   - No merge conflicts
   - Branch is fully up to date with the default branch (unless `--skip-rebase` is used with `merge`)
4. **Sequential processing** - Repositories are processed one at a time, never in parallel
5. **No local checkout** - All operations use the GitHub API

## Non-Goals

- No local git operations or repository checkouts
- No parallel repository operations
- No creating or approving pull requests
- No modifying repository settings

## Execution Flow

### merge command

```
scan → evaluate → merge → report
```

For each repository (processed sequentially):

1. Fetch repository metadata including default branch (archived repositories are skipped)
2. Enumerate candidate PRs matching `--source-branch` patterns (can be specified multiple times)
3. For each candidate PR:
   - Evaluate readiness (checks, conflicts, branch status)
   - If PR is valid: attempt merge
   - With `--skip-rebase`: attempt merge even if branch is behind
   - Record result immediately
4. Show progress bar during scanning
   - Stream each action result to the console immediately with the progress bar continuing below
   - With `--verbose`, stream each repository result as soon as it is known
   - With `--confirm`, scan without actions, prompt, then stream each action result during execution
5. Print condensed summary

### rebase command

```
scan → evaluate → rebase → report
```

For each repository (processed sequentially):

1. Fetch repository metadata including default branch (archived repositories are skipped)
2. Enumerate candidate PRs matching `--source-branch` patterns (can be specified multiple times)
3. For each candidate PR:
   - Evaluate readiness (checks, conflicts, branch status)
   - Update branch if behind
   - Record result immediately
4. Show progress bar during scanning
   - Stream each action result to the console immediately with the progress bar continuing below
   - With `--verbose`, stream each repository result as soon as it is known
   - With `--confirm`, scan without actions, prompt, then stream each action result during execution
5. Print condensed summary

### report command

```
scan → collect open PRs → group by branch name → filter → sort → display
```

1. Discover repositories using the same logic as the other subcommands
2. Collect all open, non-draft PRs targeting the default branch
3. Group PRs by exact source branch name
4. Filter by `--source-branch-prefix` if set
5. Exclude groups smaller than `--min-group-size` (default: 2)
6. Sort by descending count, then ascending branch name
7. Display results according to `--verbosity` level or as JSON with `--json`

## Contents

- [USAGE.md](USAGE.md) - Complete command-line reference with flag table
- [MERGE.md](MERGE.md) - Merge subcommand details
- [REBASE.md](REBASE.md) - Rebase subcommand details
- [REPORT.md](REPORT.md) - Report subcommand details
- [EXAMPLES.md](EXAMPLES.md) - Practical example commands and workflows
- [INSTALL.md](INSTALL.md) - Installation instructions
