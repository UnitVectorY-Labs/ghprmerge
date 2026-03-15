---
layout: default
title: ghprmerge
nav_order: 1
permalink: /
---

# ghprmerge

## Purpose

ghprmerge solves the problem of merging many similar pull requests across a GitHub organization. When you have dozens or hundreds of repositories with Dependabot (or similar automated) PRs, manually reviewing and merging each one becomes impractical.

In **report mode** (`--report`), ghprmerge scans open PRs across the organization and groups them by source branch name, helping you identify common updates that span multiple repositories.

## Safety Model

ghprmerge is designed to be **safe by default**:

1. **Default is analysis only** - Without explicit `--rebase` or `--merge` flags, the tool only scans and reports what it would do
2. **Explicit action flags** - Use `--rebase` to update branches, `--merge` to merge PRs, or `--merge --skip-rebase` to merge without requiring up-to-date branches
3. **Strict readiness checks** - A PR is only considered ready if:
   - All check runs have a successful conclusion (including non-required checks), or no checks are configured at all
   - All commit status contexts are successful, or no statuses are configured at all
   - No merge conflicts
   - Branch is fully up to date with the default branch (unless `--skip-rebase` is used)
4. **Sequential processing** - Repositories are processed one at a time, never in parallel
5. **No local checkout** - All operations use the GitHub API

## Non-Goals

- No local git operations or repository checkouts
- No parallel repository operations
- No creating or approving pull requests
- No modifying repository settings

## Execution Flow

### Normal Mode

```
scan → evaluate → optional rebase → optional merge → report
```

For each repository (processed sequentially):

1. Fetch repository metadata including default branch (archived repositories are skipped)
2. Enumerate candidate PRs matching `--source-branch` pattern
3. For each candidate PR:
   - Evaluate readiness (checks, conflicts, branch status)
   - If `--rebase`: update branch if behind
   - If `--merge` and PR is valid: attempt merge
   - If `--merge --skip-rebase`: attempt merge even if branch is behind
   - Record result immediately
4. Show progress bar during scanning
   - When an action is performed (merge or rebase), stream the result to the console immediately with the progress bar continuing below
   - In analysis mode, print matching repos after the scan completes
   - With `--verbose`, stream each repository result as soon as it is known
   - With `--confirm`, scan without actions, prompt, then stream each action result during execution
5. Print condensed summary

### Report Mode

```
scan → collect open PRs → group by branch name → filter → sort → display
```

1. Discover repositories using the same logic as normal mode
2. Collect all open, non-draft PRs targeting the default branch
3. Group PRs by exact source branch name
4. Filter by `--source-branch-prefix` if set
5. Exclude groups smaller than `--min-group-size` (default: 2)
6. Sort by descending count, then ascending branch name
7. Display results according to `--verbosity` level or as JSON with `--json`

## Contents

- [USAGE.md](USAGE.md) - Complete command-line reference with flag table
- [EXAMPLES.md](EXAMPLES.md) - Practical example commands and workflows
- [INSTALL.md](INSTALL.md) - Installation instructions
