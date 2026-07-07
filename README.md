# leftoff

[![CI](https://github.com/Halfblood-Prince/leftoff/actions/workflows/ci.yml/badge.svg)](https://github.com/Halfblood-Prince/leftoff/actions/workflows/ci.yml)
[![CodeQL](https://github.com/Halfblood-Prince/leftoff/actions/workflows/codeql.yml/badge.svg)](https://github.com/Halfblood-Prince/leftoff/actions/workflows/codeql.yml)
[![Fuzz](https://github.com/Halfblood-Prince/leftoff/actions/workflows/fuzz.yml/badge.svg)](https://github.com/Halfblood-Prince/leftoff/actions/workflows/fuzz.yml)
[![Release](https://github.com/Halfblood-Prince/leftoff/actions/workflows/release.yml/badge.svg)](https://github.com/Halfblood-Prince/leftoff/actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/go-1.25.8%2B-00ADD8)](go.mod)

**Your personal operating system for unfinished developer work.**

`leftoff` is a local-first skill and command-line tool for turning explicit development notes into an actionable work queue. The Go application is canonical at the repository root in [cmd/leftoff](cmd/leftoff) and [internal/leftoff](internal/leftoff). The plugin package in [plugins/leftoff](plugins/leftoff) is a marketplace packaging layer with Claude/Codex manifests and the self-contained [plugins/leftoff/skills/leftoff/SKILL.md](plugins/leftoff/skills/leftoff/SKILL.md).

## Install

Install the CLI with Go 1.25.8+:

```sh
go install github.com/Halfblood-Prince/leftoff/cmd/leftoff@latest
```

Install from the repo-local Codex marketplace:

```sh
codex plugin marketplace add Halfblood-Prince/leftoff
```

Install with Claude Code:

```sh
claude plugin marketplace add Halfblood-Prince/leftoff --scope user
claude plugin install leftoff@leftoff-marketplace
```

Install as a GitHub Agent Skill:

```sh
gh skill preview Halfblood-Prince/leftoff leftoff
gh skill install Halfblood-Prince/leftoff leftoff --agent codex --scope user
```

## Development

```sh
gofmt -w cmd internal
go test ./...
```

Plugin helper checks:

```sh
sh plugins/leftoff/install.sh --agent codex --dry-run
sh plugins/leftoff/scripts/check-version.sh
```

## Support Status

Officially tested:

- Claude Code
- Codex
- GitHub Agent Skills CLI

Compatible:

- Hosts that support standard `SKILL.md` folders and can load [plugins/leftoff/skills/leftoff/SKILL.md](plugins/leftoff/skills/leftoff/SKILL.md).

Experimental:

- Host-specific adapter notes included under [plugins/leftoff/agents](plugins/leftoff/agents), but not yet end-to-end tested across those hosts.

See [plugins/leftoff/agents/supported.md](plugins/leftoff/agents/supported.md) for the compatibility matrix and [plugins/leftoff/README.md](plugins/leftoff/README.md) for plugin packaging details.
