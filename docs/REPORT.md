---
layout: default
title: Report Command
nav_order: 7
permalink: /report
---

# Report Command

The `report` subcommand scans open pull requests across repositories in a GitHub organization and groups them by source branch name. It is a read-only command — no mutations are performed.

## Synopsis

```
ghprmerge [global-flags] report [report-flags]
```

The report subcommand discovers repositories using the same logic as the `merge` and `rebase` subcommands, collects all open PRs (non-draft, targeting the default branch), and groups them by exact source branch name. Groups are sorted by descending PR count, with ties broken by ascending branch name. This is useful for identifying batches of related PRs (e.g., Dependabot updates for the same dependency) that can be merged or rebased together.

## Flags

### Global Flags

These flags are placed before the `report` subcommand.

| Flag | Default | Description |
|------|---------|-------------|
| `--org` | `GITHUB_ORG` env | GitHub organization to scan (required) |
| `--repo` | - | Limit to specific repositories (repeatable) |
| `--repo-limit` | `0` | Maximum repositories to process (0 = unlimited) |
| `--json` | `false` | Output structured JSON |
| `--verbose` | `false` | Show all repos including those with no matching PRs |
| `--no-color` | `false` | Disable colored output |

### Report Flags

These flags are placed after the `report` subcommand.

| Flag | Default | Description |
|------|---------|-------------|
| `--source-branch-prefix` | - | Comma-separated list of branch prefixes to include in report |
| `--min-group-size` | `2` | Minimum number of PRs in a group to include in report |
| `--verbosity` | `standard` | Report output verbosity: `brief`, `standard`, or `verbose` |
| `--repo` | - | Additional repo filter, in addition to the global `--repo` (repeatable) |

**Flag restrictions**: The flags `--source-branch`, `--skip-rebase`, and `--confirm` cannot be used with the `report` subcommand. These flags are specific to `merge` and `rebase`.

## Behavior

The report subcommand processes repositories sequentially using the same discovery logic as the `merge` and `rebase` subcommands:

1. **Discover repositories**: Enumerate repositories in the organization, respecting `--repo` and `--repo-limit` filters. Archived repositories are excluded.
2. **Collect open PRs**: For each repository, list all open pull requests that are not drafts and target the default branch.
3. **Group by source branch**: Group collected PRs by their exact head branch name.
4. **Filter by prefix**: If `--source-branch-prefix` is set, only groups whose branch name starts with one of the specified prefixes are included.
5. **Filter by group size**: Groups with fewer PRs than `--min-group-size` (default: 2) are excluded.
6. **Sort**: Groups are sorted by descending PR count. Ties are broken by ascending branch name.
7. **Evaluate status**: Each PR's status is evaluated using the same logic as the `merge` and `rebase` subcommands (check status, branch status relative to default branch).

**No mutations are performed.** The report subcommand is entirely read-only. It never merges, rebases, or comments on pull requests.

## Verbosity

The `--verbosity` flag controls the level of detail in text output. It accepts three values:

### `brief`

Displays only the branch name and PR count for each group:

```
dependabot/go_modules/foo-1.2.3 (3 PRs)
dependabot/npm_and_yarn/bar-2.0.0 (2 PRs)
```

### `standard` (default)

Displays the branch name, count, and for each PR: the repository name, PR number, and status:

```
dependabot/go_modules/foo-1.2.3 (3 PRs)
  repo-a     #123  passing
  repo-b     #456  needs-rebase
  repo-c     #789  checks pending

dependabot/npm_and_yarn/bar-2.0.0 (2 PRs)
  repo-a     #124  passing
  repo-d     #321  conflict
```

### `verbose`

Includes everything from `standard` plus the PR title:

```
dependabot/go_modules/foo-1.2.3 (3 PRs)
  repo-a     #123  passing         Bump foo from 1.2.2 to 1.2.3
  repo-b     #456  needs-rebase    Bump foo from 1.2.2 to 1.2.3
  repo-c     #789  checks pending  Bump foo from 1.2.2 to 1.2.3

dependabot/npm_and_yarn/bar-2.0.0 (2 PRs)
  repo-a     #124  passing         Bump bar from 1.9.0 to 2.0.0
  repo-d     #321  conflict        Bump bar from 1.9.0 to 2.0.0
```

The `--verbosity` flag only affects text output. JSON output always includes all fields regardless of the verbosity setting.

## JSON Output

With `--json`, the report subcommand outputs structured JSON. The schema is the same regardless of `--verbosity`:

```json
{
  "groups": [
    {
      "sourceBranch": "dependabot/go_modules/foo-1.2.3",
      "count": 3,
      "pullRequests": [
        {
          "repository": "repo-a",
          "number": 123,
          "status": "passing",
          "title": "Bump foo from 1.2.2 to 1.2.3",
          "url": "https://github.com/myorg/repo-a/pull/123"
        },
        {
          "repository": "repo-b",
          "number": 456,
          "status": "needs-rebase",
          "title": "Bump foo from 1.2.2 to 1.2.3",
          "url": "https://github.com/myorg/repo-b/pull/456"
        },
        {
          "repository": "repo-c",
          "number": 789,
          "status": "checks pending",
          "title": "Bump foo from 1.2.2 to 1.2.3",
          "url": "https://github.com/myorg/repo-c/pull/789"
        }
      ]
    }
  ]
}
```

Each group contains:

| Field | Type | Description |
|-------|------|-------------|
| `sourceBranch` | string | The exact head branch name shared by PRs in this group |
| `count` | number | The number of PRs in the group |
| `pullRequests` | array | List of PRs in the group |

Each pull request contains:

| Field | Type | Description |
|-------|------|-------------|
| `repository` | string | The repository name (without the organization prefix) |
| `number` | number | The pull request number |
| `status` | string | The evaluated status of the PR (see [Status Values](#status-values)) |
| `title` | string | The pull request title |
| `url` | string | The full URL to the pull request on GitHub |

## Status Values

Each PR is assigned one of the following status values, using the same evaluation logic as the `merge` and `rebase` subcommands:

| Status | Description |
|--------|-------------|
| `passing` | All checks passing and branch is up-to-date with the default branch |
| `needs-rebase` | Branch is behind the default branch |
| `conflict` | PR has merge conflicts |
| `checks failing` | One or more required checks have failed |
| `checks pending` | Checks are still running |
| `no checks configured` | No status checks are configured for the repository |
| `error` | An error occurred while evaluating the PR |

## Empty Results

When no grouped source branches match the filters:

- **Text output**: Displays `No grouped source branches found.`
- **JSON output**: Returns `{"groups": []}`
- **Exit code**: `0` — empty results are not an error

This can happen when:

- No open PRs exist in the organization
- All PRs have unique source branch names and `--min-group-size` filters them out
- The `--source-branch-prefix` filter does not match any branch names
- The `--repo` filter limits discovery to repositories with no matching PRs

## Examples

### Basic report

Scan the organization and show all grouped PRs:

```bash
ghprmerge --org myorg report
```

### Filter by branch prefix

Show only Dependabot Go module updates:

```bash
ghprmerge --org myorg report --source-branch-prefix dependabot/go_modules/
```

### Filter by multiple prefixes

Show Dependabot updates for both Go and npm:

```bash
ghprmerge --org myorg report --source-branch-prefix dependabot/go_modules/,dependabot/npm_and_yarn/
```

### Include single-PR groups

Lower the minimum group size to include branches with only one PR:

```bash
ghprmerge --org myorg report --min-group-size 1
```

### Brief output

Show only branch names and counts:

```bash
ghprmerge --org myorg report --verbosity brief
```

### Verbose output

Show branch names, counts, statuses, and PR titles:

```bash
ghprmerge --org myorg report --verbosity verbose
```

### Limit to specific repos

Report on specific repositories only:

```bash
ghprmerge --org myorg --repo repo-a --repo repo-b report
```

### JSON output for scripting

Get structured output for automation pipelines:

```bash
ghprmerge --org myorg --json report | jq '.groups[] | select(.count >= 5)'
```

### Combine with repo limit

Scan at most 20 repositories:

```bash
ghprmerge --org myorg --repo-limit 20 report
```

### Disable colored output

Useful for CI environments or piping to a file:

```bash
ghprmerge --org myorg --no-color report
```

### Identify PRs ready to merge

Use JSON output to find groups where all PRs are passing:

```bash
ghprmerge --org myorg --json report | jq '
  .groups[] | select(
    all(.pullRequests[]; .status == "passing")
  )
'
```

### Feed report into merge

Use the report to identify a branch, then merge it:

```bash
# Step 1: Find grouped branches
ghprmerge --org myorg report --verbosity brief

# Step 2: Merge a specific branch from the report
ghprmerge --org myorg merge --source-branch dependabot/go_modules/foo-1.2.3
```
