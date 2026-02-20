---
layout: default
title: Install
nav_order: 2
permalink: /install
---

# Installation

## Prerequisites

- A GitHub personal access token with appropriate permissions, or
- GitHub CLI (`gh`) installed and authenticated

## Binary Download

Download the latest release from the [releases page](https://github.com/UnitVectorY-Labs/ghprmerge/releases).

### Linux (amd64)

```bash
curl -L https://github.com/UnitVectorY-Labs/ghprmerge/releases/latest/download/ghprmerge_linux_amd64 -o ghprmerge
chmod +x ghprmerge
sudo mv ghprmerge /usr/local/bin/
```

### macOS (amd64)

```bash
curl -L https://github.com/UnitVectorY-Labs/ghprmerge/releases/latest/download/ghprmerge_darwin_amd64 -o ghprmerge
chmod +x ghprmerge
sudo mv ghprmerge /usr/local/bin/
```

### macOS (arm64 / Apple Silicon)

```bash
curl -L https://github.com/UnitVectorY-Labs/ghprmerge/releases/latest/download/ghprmerge_darwin_arm64 -o ghprmerge
chmod +x ghprmerge
sudo mv ghprmerge /usr/local/bin/
```

### Windows

Download `ghprmerge_windows_amd64.exe` from the releases page and add it to your PATH.

## Building from Source

### Build

```bash
git clone https://github.com/UnitVectorY-Labs/ghprmerge.git
cd ghprmerge
go build -o ghprmerge .
```

### Install to GOPATH

```bash
go install github.com/UnitVectorY-Labs/ghprmerge@latest
```

## Verify Installation

```bash
ghprmerge --help
```

## Authentication Setup

### Using GITHUB_TOKEN

```bash
export GITHUB_TOKEN=ghp_xxxxxxxxxxxx
ghprmerge --org myorg --source-branch dependabot/
```

### Using GitHub CLI

```bash
gh auth login
ghprmerge --org myorg --source-branch dependabot/
```

The tool will automatically use the token from `gh auth token` if `GITHUB_TOKEN` is not set.
