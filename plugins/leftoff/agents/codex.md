# Codex Adapter

Status: Supported
Installer aliases: `codex`
Suggested target: `${CODEX_HOME:-~/.codex}/skills/leftoff/`

Install from the repo-local Codex marketplace, or symlink the repository into a local plugin source, then let Codex load `skills/leftoff/SKILL.md` as the canonical instruction file.

Use `.codex-plugin/plugin.json` as the Codex plugin manifest. `agents/openai.yaml` is legacy adapter metadata only; do not treat it as packaging metadata. Do not change the shared behavior in this adapter; update `skills/leftoff/SKILL.md` or `skills/leftoff/references/` instead.
