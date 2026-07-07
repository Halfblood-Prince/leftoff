# Hermes Agent Adapter

Status: Experimental
Installer aliases: `hermes-agent`, `hermes`
Suggested target: `~/.hermes/skills/leftoff/`

Use this adapter for Hermes-based agents that support reusable local instructions or skill folders.

If Hermes uses a workspace-specific location, copy or symlink the repository there and load `skills/leftoff/SKILL.md` first. The adapter should only map host loading behavior; it should not redefine storage or safety rules.
