# Examples

## Basic Dry Run

Preview what would happen without making any changes:

```bash
ghprmerge --org myorg --source-branch dependabot/
```

This scans all repositories in `myorg` for open pull requests with branches containing `dependabot/` and reports what would be merged.

## Merge Dependabot Pull Requests

Merge all ready Dependabot pull requests:

```bash
ghprmerge --org myorg --source-branch dependabot/ --dry-run=false
```

## Merge with Auto-Rebase

Automatically update out-of-date branches before merging:

```bash
ghprmerge --org myorg --source-branch dependabot/ --dry-run=false --rebase
```

For Dependabot PRs, this posts a `@dependabot rebase` comment. For other branches, it uses GitHub's update branch API.

## Limit Number of Merges

Merge at most 10 pull requests (useful for gradual rollout):

```bash
ghprmerge --org myorg --source-branch dependabot/ --dry-run=false --limit 10
```

## Scope to Specific Repositories

Only process specific repositories:

```bash
ghprmerge --org myorg --source-branch dependabot/ --repo repo1 --repo repo2
```

## JSON Output

Get structured JSON output for scripting:

```bash
ghprmerge --org myorg --source-branch dependabot/ --json
```

## Using Environment Variables

Set the organization via environment variable:

```bash
export GITHUB_ORG=myorg
ghprmerge --source-branch dependabot/
```

## Complete Production Example

A complete example for merging Dependabot PRs across an organization:

```bash
# Set authentication
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx

# Dry run first to preview
ghprmerge --org myorg --source-branch dependabot/

# If everything looks good, perform actual merges
ghprmerge --org myorg --source-branch dependabot/ --dry-run=false --rebase --limit 50
```

## Filtering by Package Ecosystem

Match specific package managers:

```bash
# Only npm updates
ghprmerge --org myorg --source-branch dependabot/npm_and_yarn/

# Only Go module updates  
ghprmerge --org myorg --source-branch dependabot/go_modules/

# Only Maven updates
ghprmerge --org myorg --source-branch dependabot/maven/
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
      - name: Merge Dependabot PRs
        env:
          GITHUB_TOKEN: ${{ secrets.GH_PAT }}
        run: |
          # Download ghprmerge
          curl -L https://github.com/UnitVectorY-Labs/ghprmerge/releases/latest/download/ghprmerge_linux_amd64 -o ghprmerge
          chmod +x ghprmerge
          
          # Preview first
          ./ghprmerge --org myorg --source-branch dependabot/ --json
          
          # Merge with limit
          ./ghprmerge --org myorg --source-branch dependabot/ --dry-run=false --rebase --limit 20
```
