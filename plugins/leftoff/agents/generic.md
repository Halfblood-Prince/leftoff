# Generic Agent Adapter

Status: Supported
Installer aliases: `generic`
Suggested target: `~/.leftoff/skills/leftoff/`

Load `skills/leftoff/SKILL.md` as the primary instruction file for any agent that supports custom skills, tools, memory packs, or reusable instructions.

Use the plugin launcher for deterministic local actions. Before running a command that writes records, state which store path will be used and what kind of metadata will be saved.

If the host has its own skill directory, copy or symlink the repository there and keep `skills/leftoff/SKILL.md`, `skills/leftoff/references/`, `skills/leftoff/templates/`, `bin/`, `cmd/`, and `internal/` together.
