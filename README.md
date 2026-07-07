# leftoff

[![CI](actions/workflows/ci.yml/badge.svg)](actions/workflows/ci.yml)
[![CodeQL](actions/workflows/codeql.yml/badge.svg)](actions/workflows/codeql.yml)
[![Fuzz](actions/workflows/fuzz.yml/badge.svg)](actions/workflows/fuzz.yml)
[![Release](actions/workflows/release.yml/badge.svg)](actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8)](plugins/leftoff/go.mod)

**Your personal operating system for unfinished developer work.**

`leftoff` is a local-first skill and command-line tool for turning explicit development notes into an actionable work queue. The canonical product package lives in [plugins/leftoff](plugins/leftoff), which contains the Claude and Codex plugin manifests, `skills/leftoff/SKILL.md`, launchers, Go source, tests, setup scripts, adapter notes, and release assets.

The root of this repository only contains marketplace catalogues, workflows, project docs, and the plugin package. Keeping the implementation under `plugins/leftoff` avoids publishing two discoverable copies of the same skill.

## Installation

Install from the repo-local Codex marketplace:

```sh
codex plugin marketplace add Halfblood-Prince/leftoff
```

Restart Codex, then select and install `leftoff` from the Plugins directory.

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

For local development or manual folder installs:

```sh
cd plugins/leftoff
./install.sh --agent codex --dry-run
./install.sh --agent claude --mode symlink
go test ./...
```

## Support Status

Officially tested:

- Claude Code
- Codex
- GitHub Agent Skills CLI

Compatible:

- Hosts that support standard `SKILL.md` folders and can load `skills/leftoff/SKILL.md`.

Experimental:

- Host-specific adapter notes included under [plugins/leftoff/agents](plugins/leftoff/agents), but not yet end-to-end tested across those hosts.

See [plugins/leftoff/README.md](plugins/leftoff/README.md) for the full user guide, privacy model, binary setup flow, command reference, and development notes.
