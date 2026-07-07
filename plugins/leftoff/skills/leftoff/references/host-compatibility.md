# Agent Compatibility

`leftoff` is a host-neutral skill backed by a Go CLI. Hosts should load `skills/leftoff/SKILL.md` for shared behavior and use files in `agents/` only for host-specific loading details.

## Requirements

- The plugin launcher at `bin/leftoff` or `bin/leftoff.ps1`.
- Go 1.22+ only to build, test, or use the launcher source fallback.
- Git is optional. Without Git, capture and validation still work; resume reports the missing Git context as uncertainty.
- The core workflow does not require network access.
- Binary setup requires explicit user approval before network access and verifies GitHub artifact provenance plus `SHA256SUMS`.

## Agent Targets

The complete compatibility matrix is maintained in [../../../agents/supported.md](../../../agents/supported.md). It includes:

- status for each requested agent;
- installer aliases such as `claude-code`, `cursor`, `copilot-cli`, `gemini`, `antigravity`, `roo-code`, and `windsurf`;
- the adapter file for each agent;
- a suggested local install target.

Suggested targets are local conventions. If an agent uses a different custom-instruction directory, use `--target <path>` and make `skills/leftoff/SKILL.md` the entry point.

## Compatibility Matrix

| Capability | Standard skill hosts |
|---|---|
| Load `skills/leftoff/SKILL.md` | yes |
| Run plugin launcher through shell | yes |
| Local store under `~/.leftoff` | yes |
| Core workflow without network | yes |
| Optional GitHub metadata | opt-in `gh` |
| Host-specific APIs required | no |

## Installation

Dry run:

```sh
./install.sh --agent codex --dry-run
```

Copy install:

```sh
./install.sh --agent cursor --mode copy
```

Symlink install:

```sh
./install.sh --agent claude-code --mode symlink
```

Custom target:

```sh
./install.sh --target "$HOME/.custom-agent/skills/leftoff" --mode copy
```

PowerShell:

```powershell
.\install.ps1 -Agent windsurf -DryRun
.\install.ps1 -Agent openclaw -Mode copy
.\install.ps1 -Target "$HOME\.custom-agent\skills\leftoff" -Mode copy
```

## Core Commands

```sh
./bin/leftoff init
./bin/leftoff capture "task: ..."
./bin/leftoff now --minutes 45
./bin/leftoff scan --repo .
./bin/leftoff resume --repo .
./bin/leftoff clean-up
./bin/leftoff compat
./bin/leftoff validate --repair
```

See [manual-smoke-test.md](manual-smoke-test.md) for a clean-store end-to-end checklist.
