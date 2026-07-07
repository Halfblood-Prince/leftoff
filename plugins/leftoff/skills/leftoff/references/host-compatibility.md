# Agent Compatibility

`leftoff` is a host-neutral skill backed by a Go CLI. Hosts should load this `SKILL.md` for shared behavior and use the local `scripts/` launchers for command execution.

## Requirements

- The skill launcher at `scripts/leftoff` or `scripts/leftoff.ps1`.
- Go 1.22+ only to build, test, or install the CLI with `go install`.
- Git is optional. Without Git, capture and validation still work; resume reports the missing Git context as uncertainty.
- The core workflow does not require network access.
- Binary setup requires explicit user approval before network access and verifies GitHub artifact provenance plus `SHA256SUMS`.

## Agent Targets

The repository compatibility matrix is maintained outside this self-contained skill package. This skill itself includes:

- a host-neutral `SKILL.md`;
- `scripts/leftoff` and `scripts/leftoff.ps1` launchers;
- setup scripts for installing a verified local binary;
- references and templates needed by the skill.

Suggested targets are local conventions. If an agent uses a different custom-instruction directory, copy or install this skill directory there and make `SKILL.md` the entry point.

## Compatibility Matrix

| Capability | Standard skill hosts |
|---|---|
| Load `SKILL.md` | yes |
| Run skill launcher through shell | yes |
| Local store under `~/.leftoff` | yes |
| Core workflow without network | yes |
| Optional GitHub metadata | opt-in `gh` |
| Host-specific APIs required | no |

## Installation

Install this skill by copying or installing the directory that contains this file:

- `SKILL.md`
- `scripts/`
- `references/`
- `templates/`

Repository and plugin marketplace installs may provide additional plugin-level manifests and helper installers, but the skill directory above is enough for GitHub Agent Skills CLI installs.

## Core Commands

```sh
./scripts/leftoff init
./scripts/leftoff capture "task: ..."
./scripts/leftoff now --minutes 45
./scripts/leftoff scan --repo .
./scripts/leftoff resume --repo .
./scripts/leftoff clean-up
./scripts/leftoff compat
./scripts/leftoff validate --repair
```

See [manual-smoke-test.md](manual-smoke-test.md) for a clean-store end-to-end checklist.
