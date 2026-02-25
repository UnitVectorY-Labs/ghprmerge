---
layout: default
title: ghprmerge
nav_order: 1
permalink: /
---

# ghprmerge

## Purpose

ghprmerge solves the problem of merging many similar pull requests across a GitHub organization. When you have dozens or hundreds of repositories with Dependabot (or similar automated) PRs, manually reviewing and merging each one becomes impractical.

## Safety Model

ghprmerge is designed to be **safe by default**:

1. **Default is analysis only** - Without explicit `--rebase` or `--merge` flags, the tool only scans and reports what it would do
2. **Explicit action flags** - Use `--rebase` to update branches, `--merge` to merge PRs, or `--merge --skip-rebase` to merge without requiring up-to-date branches
3. **Strict readiness checks** - A PR is only considered ready if:
   - All check runs have a successful conclusion (including non-required checks)
   - All commit status contexts are successful
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
4. Print per-repository summary

## Contents

- [USAGE.md](USAGE.md) - Complete command-line reference with flag table
- [EXAMPLES.md](EXAMPLES.md) - Practical example commands and workflows
- [INSTALL.md](INSTALL.md) - Installation instructions
