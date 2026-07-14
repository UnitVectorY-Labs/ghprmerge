---
name: ghprmerge
description: >
  Use this skill when you need to batch-evaluate, rebase, or merge GitHub pull
  requests across multiple repositories in a GitHub organization. Useful for
  managing automated dependency updates (e.g., Dependabot) at scale.
license: MIT
compatibility: standalone binary; requires GitHub API access via GITHUB_TOKEN or gh auth
---

# ghprmerge

Batch-manage pull requests across a GitHub organization via the GitHub API. No local checkouts.

## Use this skill when
- The user wants to merge or rebase PRs across many repositories in a GitHub org.
- The user needs a report of open PRs grouped by branch name across an organization.
- The user mentions "batch merging", "bulk rebase", or "Dependabot management" for an organization.

## Do not use this skill when
- The task involves a single PR in a single repository (use `gh pr merge`).
- The task requires local git operations or manual code review.

## First checks
1. **Installation**: Run `ghprmerge --version`.
2. **Authentication**: Check for `GITHUB_TOKEN` env var or run `gh auth status`.
3. **Context**: Ensure `--org` flag or `GITHUB_ORG` env var is provided.

## Command Reference

### Global flags (before subcommand)
| Flag | Default | Purpose |
|---|---|---|
| `--org` | `GITHUB_ORG` env | GitHub org to scan (required) |
| `--repo` | — | Limit to specific repos (repeatable) |
| `--repo-limit` | `0` | Max repos to process |
| `--json` | `false` | Structured JSON output |
| `--verbose` | `false` | Show all repos including those with no matching PRs |
| `--no-color` | `false` | Disable ANSI colors |
| `--no-progress` | `false` | Suppress progress bar |
| `--version` | — | Show version and exit |

### `merge` — merge ready PRs
`ghprmerge --org <org> merge --source-branch <pattern> [flags]`

| Flag | Default | Purpose |
|---|---|---|
| `--source-branch` | — | Branch pattern to match (required, repeatable, substring match) |
| `--skip-rebase` | `false` | Merge PRs even if they are behind the default branch |
| `--confirm` | `false` | Scan and prompt for confirmation before merging |
| `--repo` | — | Additional repo filter (repeatable) |

### `rebase` — update stale PRs
`ghprmerge --org <org> rebase --source-branch <pattern> [flags]`

| Flag | Default | Purpose |
|---|---|---|
| `--source-branch` | — | Branch pattern to match (required, repeatable, substring match) |
| `--confirm` | `false` | Scan and prompt for confirmation before rebasing |
| `--repo` | — | Additional repo filter (repeatable) |

### `report` — read-only overview
`ghprmerge --org <org> report [flags]`

| Flag | Default | Purpose |
|---|---|---|
| `--source-branch-prefix` | — | Comma-separated branch prefixes (prefix match) |
| `--min-group-size` | `2` | Min PRs per group to include |
| `--verbosity` | `standard` | `brief`, `standard`, or `verbose` |
| `--repo` | — | Additional repo filter (repeatable) |

**Note**: `report` uses `--source-branch-prefix` (prefix), not `--source-branch` (substring). `--skip-rebase` and `--confirm` are NOT valid with `report`.

## Analysis-only mode
Running `ghprmerge --org <org>` without a subcommand but with `--source-branch` evaluates PRs and shows results without performing mutations.

## Tool Behavior
- **Matching**: `--source-branch` uses substring matching (e.g., `dependabot/` matches `dependabot/npm_and_yarn/foo`).
- **Filtering**: Draft PRs, PRs not targeting the default branch, and non-matching patterns are silently filtered.
- **Merge Logic**: Pending checks block merge. PRs with no configured checks proceed.
- **Rebase Logic**: Rebase does not block on failing checks.
- **Repo Limit**: `--repo-limit` marks remaining repos as skipped in merge/rebase; in report, they are silently dropped.

## Skip reasons
| Reason | Meaning |
|---|---|
| `merge conflict` | PR has merge conflicts |
| `checks failing` | A check run or status has failed |
| `checks pending` | A check is still running |
| `branch behind default` | Branch is behind default (ignored if `--skip-rebase` is used) |
| `API error` | GitHub API returned an error |
| `repo limit reached` | Skipped due to `--repo-limit` |

## Output & Troubleshooting
- **JSON**: Use `--json` for programmatic processing.
- **Auth**: Verify `GITHUB_TOKEN` or `gh auth status`.
- **Rate Limits**: Use `--repo-limit` to throttle requests.
- **Empty Results**: Exit code 0 with no results means no PRs matched.
