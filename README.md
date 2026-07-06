# leftoff

[![CI](../../actions/workflows/ci.yml/badge.svg)](../../actions/workflows/ci.yml)
[![CodeQL](../../actions/workflows/codeql.yml/badge.svg)](../../actions/workflows/codeql.yml)
[![Fuzz](../../actions/workflows/fuzz.yml/badge.svg)](../../actions/workflows/fuzz.yml)
[![Release](../../actions/workflows/release.yml/badge.svg)](../../actions/workflows/release.yml)
[![Go](https://img.shields.io/badge/go-1.22%2B-00ADD8)](go.mod)

**Your personal operating system for unfinished developer work.**

`leftoff` is a local-first skill and command-line tool for turning explicit development notes into an actionable work queue. It is designed for AI coding agents and developer assistants across many local custom-instruction hosts.

It stores durable records as Markdown and JSONL under `~/.leftoff/`, so everything remains inspectable, editable, portable, and easy to delete.

The project is written in Go and uses only the Go standard library.

## What It Does

- Provides a shared `skills/leftoff/SKILL.md` contract for AI agents.
- Ships Claude and Codex plugin manifests plus repo-local marketplace catalogues.
- Includes lightweight adapter notes under `agents/`.
- Initializes and validates a plain-text local store.
- Captures tasks, ideas, decisions, problems, solutions, releases, and follow-ups.
- Rejects likely secrets before writing records.
- Saves compact, read-only Git state from a repository you choose.
- Tracks a registered workspace of repositories with safe Git metadata only.
- Rebuilds project context with `resume`.
- Recommends next work with explainable `now` scoring.
- Emits structured JSON for agent-facing `now`, `resume`, `scan`, and `github` commands.
- Recalls decisions and solved problems.
- Produces weekly reviews and recurring-friction reports.
- Reports cleanup opportunities without deleting Git branches or worktrees.
- Supports export, import, and guarded local-data deletion.

The core workflow does not require network access, analytics, a GitHub token, or any external service. GitHub metadata is optional and runs only when explicitly requested with `github --refresh`.

## Supported Agents

The shared behavior lives in [skills/leftoff/SKILL.md](skills/leftoff/SKILL.md). The full support matrix, aliases, adapter files, and suggested local targets live in [agents/supported.md](agents/supported.md).

Supported agents include Claude Code, Codex, Cursor, Pi, GitHub Copilot CLI, OpenCode, Gemini CLI / Antigravity, Factory AI Droid, OpenClaw, Hermes Agent, AstrBot, NanoClaw, Shelley, Auggie / Augment, Cline / Roo Code, CodeBuddy, Continue, Crush, Deep Agents, Firebender, ForgeCode, Goose, Junie, Kilo Code, Kimi Code CLI, Kiro CLI, Lingma, Mistral Vibe, Mux, OpenHands, Qoder, Qwen Code, Rovo Dev, Tabnine CLI, Trae / Trae CN, Warp, Windsurf, Zed, and generic Markdown-instruction hosts.

## Installation

Install from the repo-local Codex marketplace:

```sh
codex plugin marketplace add Halfblood-Prince/leftoff
```

Restart Codex, then select and install `leftoff` from the Plugins directory. The Codex marketplace points at the dedicated plugin package in `plugins/leftoff/`.

Install with Claude Code:

```sh
claude plugin marketplace add Halfblood-Prince/leftoff --scope user
claude plugin install leftoff@leftoff-marketplace
```

Validate the local Claude plugin package:

```sh
cd plugins/leftoff
claude plugin validate .
```

Install as a GitHub Agent Skill:

```sh
gh skill preview Halfblood-Prince/leftoff leftoff
gh skill install Halfblood-Prince/leftoff leftoff@v0.2.0 --agent codex --scope user
```

Release bundles include `leftoff.skill.zip`, a plugin-shaped archive with `.claude-plugin/`, `.codex-plugin/`, `skills/leftoff/SKILL.md`, source, launchers, and prebuilt platform binaries where available. Platform bundles also include a single prebuilt `leftoff` binary, including macOS Intel and Apple Silicon builds.

Preview an install:

```sh
./install.sh --agent codex --dry-run
```

Install for a specific agent:

```sh
./install.sh --agent claude --mode symlink
./install.sh --agent gemini --mode copy
./install.sh --agent cursor --mode copy
```

PowerShell:

```powershell
.\install.ps1 -Agent codex -DryRun
.\install.ps1 -Agent hermes -Mode copy
.\install.ps1 -Agent windsurf -Mode copy
```

Use `--target <path>` when an agent has a custom skill directory.

## Binary Setup

Marketplace installation copies plugin source; it does not automatically fetch GitHub Release assets. The launchers live at:

```text
bin/leftoff
powershell -ExecutionPolicy Bypass -File .\bin\leftoff.ps1
```

If no bundled binary is present, ask the user before any network access, then run the setup script:

```sh
./scripts/setup-binary.sh
```

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\setup-binary.ps1
```

The setup flow downloads the platform release bundle with GitHub CLI, verifies GitHub artifact provenance, checks `SHA256SUMS`, and installs the binary under `bin/.leftoff/<os>_<arch>/`. It never uses `curl | sh`.

## Commands

Run from the plugin root:

```sh
./bin/leftoff init
./bin/leftoff capture "task: Add the Windows installation smoke test"
./bin/leftoff capture --project sample-app "decision: Use Markdown and JSONL because records must stay editable"
./bin/leftoff now --minutes 45 --focus release
./bin/leftoff scan --repo .
./bin/leftoff resume --repo .
./bin/leftoff workspace add .
./bin/leftoff workspace scan
./bin/leftoff now --all --minutes 45
./bin/leftoff remember-why --project sample-app "windows install"
./bin/leftoff save-solution --project sample-app --problem "Windows install failed" --solution "Use the documented installer path" --confirm
./bin/leftoff review-week --write
./bin/leftoff friction
./bin/leftoff clean-up --repo .
./bin/leftoff github --repo . --refresh
./bin/leftoff compat
./bin/leftoff export --out leftoff-export.zip
./bin/leftoff import --from leftoff-export.zip --confirm
./bin/leftoff delete-data --dry-run
./bin/leftoff validate --repair
```

Use `--json` with `now`, `resume`, `scan`, and `github` when an agent needs structured output:

```sh
./bin/leftoff now --all --minutes 45 --json
./bin/leftoff scan --repo . --json
./bin/leftoff resume --repo . --json
./bin/leftoff github --repo . --json
```

Use `--store <path>` in tests or experiments to avoid writing to the default `~/.leftoff/` store.

## Privacy Model

`leftoff` saves durable records only when the user runs a command that clearly writes metadata. It rejects likely secrets before persistence, avoids full diffs and source contents, and stores compact Git metadata only: dirty state, branch, head, worktree path, ahead/behind status, unpushed commit count, changed file paths, recent commit titles, stale branch names, worktree status, saved leftoff records, and sanitized remote URL.

Remote integrations are optional. Ranking, recall, review, cleanup, and friction reports use local Markdown/JSONL records unless the user explicitly runs `github --refresh`.

Cleanup is report-only by default. `leftoff` never deletes Git branches or worktrees. The only automated cleanup currently supported is exact duplicate activity-line removal with `clean-up --apply --confirm --action dedupe-activity`, and it creates a backup first.

## Data Layout

The store follows the stable layout documented in [skills/leftoff/references/record-schema.md](skills/leftoff/references/record-schema.md):

```text
~/.leftoff/
|-- profile.md
|-- config.yml
|-- inbox.md
|-- projects/<project-slug>/
|-- patterns/
|-- weekly/
|-- workspace/repos.json
|-- cache/
|   |-- workspace-scan.json
|   `-- github/<project-slug>.json
`-- backups/
```

Project task-like records are kept in `open-loops.md`; decisions, solved problems, releases, and friction each have their own Markdown files. Machine-readable activity events are append-only JSONL records.

## Development

The project has no third-party dependencies.

```sh
gofmt -w cmd internal
go test ./...
go vet ./...
govulncheck ./...
staticcheck ./...
```

CI runs the Go test suite on Linux, macOS, and Windows. Release builds include Linux, macOS, and Windows bundles for amd64 and arm64 where Go supports the target.

If Go is not installed, the source can still be inspected as plain text, but tests, formatting, and the launcher source fallback require a local Go toolchain.

## Release References

- [skills/leftoff/references/data-format.md](skills/leftoff/references/data-format.md)
- [skills/leftoff/references/export-import.md](skills/leftoff/references/export-import.md)
- [skills/leftoff/references/threat-model.md](skills/leftoff/references/threat-model.md)
- [skills/leftoff/references/github-integration.md](skills/leftoff/references/github-integration.md)
- [skills/leftoff/references/delete-data.md](skills/leftoff/references/delete-data.md)
- [skills/leftoff/references/manual-smoke-test.md](skills/leftoff/references/manual-smoke-test.md)
