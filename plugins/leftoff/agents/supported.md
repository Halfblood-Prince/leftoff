# Agent Compatibility

Suggested targets are local conventions for installation. If an agent uses a different custom-instruction directory, pass `--target <path>` to the installer and make `skills/leftoff/SKILL.md` the entry point.

## Status Meanings

- Officially tested: end-to-end install or validation is covered by project CI or release smoke tests.
- Compatible: expected to work with hosts that support standard `SKILL.md` folders, but no host-specific end-to-end test is claimed.
- Experimental: adapter notes are included, but the host has not yet been end-to-end tested by this project.

## Officially Tested

| Host | Status | Installer aliases | Adapter | Suggested target |
|---|---|---|---|---|
| Claude Code | Officially tested | `claude-code`, `claude` | `agents/claude.md` | `~/.claude/skills/leftoff/` |
| Codex | Officially tested | `codex` | `agents/codex.md` | `${CODEX_HOME:-~/.codex}/skills/leftoff/` |
| GitHub Agent Skills CLI | Officially tested | n/a | n/a | discovers `skills/leftoff/SKILL.md` |

## Compatible

| Host | Status | Installer aliases | Adapter | Suggested target |
|---|---|---|---|---|
| Generic standard `SKILL.md` host | Compatible | `generic` | `agents/generic.md` | custom skill directory |

## Experimental Adapter Notes

| Host | Status | Installer aliases | Adapter | Suggested target |
|---|---|---|---|---|
| Cursor | Experimental | `cursor` | `agents/cursor.md` | `~/.cursor/skills/leftoff/` |
| Pi | Experimental | `pi` | `agents/pi.md` | `~/.pi/skills/leftoff/` |
| GitHub Copilot CLI | Experimental | `github-copilot-cli`, `copilot-cli`, `copilot` | `agents/github-copilot-cli.md` | `~/.github-copilot-cli/skills/leftoff/` |
| OpenCode | Experimental | `opencode` | `agents/opencode.md` | `~/.opencode/skills/leftoff/` |
| Gemini CLI / Antigravity | Experimental | `gemini-cli-antigravity`, `gemini`, `google-gemini`, `antigravity` | `agents/google-gemini.md` | `~/.gemini/skills/leftoff/` |
| Factory AI Droid | Experimental | `factory-ai-droid`, `factory-droid`, `droid` | `agents/factory-ai-droid.md` | `~/.factory-ai-droid/skills/leftoff/` |
| OpenClaw | Experimental | `openclaw` | `agents/openclaw.md` | `~/.openclaw/skills/leftoff/` |
| Hermes Agent | Experimental | `hermes-agent`, `hermes` | `agents/hermes.md` | `~/.hermes/skills/leftoff/` |
| AstrBot | Experimental | `astrbot` | `agents/astrbot.md` | `~/.astrbot/skills/leftoff/` |
| NanoClaw | Experimental | `nanoclaw` | `agents/nanoclaw.md` | `~/.nanoclaw/skills/leftoff/` |
| Shelley | Experimental | `shelley` | `agents/shelley.md` | `~/.shelley/skills/leftoff/` |
| Auggie / Augment | Experimental | `auggie-augment`, `auggie`, `augment` | `agents/auggie-augment.md` | `~/.augment/skills/leftoff/` |
| Cline / Roo Code | Experimental | `cline-roo-code`, `cline`, `roo`, `roo-code` | `agents/cline-roo-code.md` | `~/.cline/skills/leftoff/` |
| CodeBuddy | Experimental | `codebuddy` | `agents/codebuddy.md` | `~/.codebuddy/skills/leftoff/` |
| Continue | Experimental | `continue` | `agents/continue.md` | `~/.continue/skills/leftoff/` |
| Crush | Experimental | `crush` | `agents/crush.md` | `~/.crush/skills/leftoff/` |
| Deep Agents | Experimental | `deep-agents`, `deepagents` | `agents/deep-agents.md` | `~/.deep-agents/skills/leftoff/` |
| Firebender | Experimental | `firebender` | `agents/firebender.md` | `~/.firebender/skills/leftoff/` |
| ForgeCode | Experimental | `forgecode` | `agents/forgecode.md` | `~/.forgecode/skills/leftoff/` |
| Goose | Experimental | `goose` | `agents/goose.md` | `~/.goose/skills/leftoff/` |
| Junie | Experimental | `junie` | `agents/junie.md` | `~/.junie/skills/leftoff/` |
| Kilo Code | Experimental | `kilo-code`, `kilocode` | `agents/kilo-code.md` | `~/.kilo-code/skills/leftoff/` |
| Kimi Code CLI | Experimental | `kimi-code-cli`, `kimi` | `agents/kimi-code-cli.md` | `~/.kimi-code-cli/skills/leftoff/` |
| Kiro CLI | Experimental | `kiro-cli`, `kiro` | `agents/kiro-cli.md` | `~/.kiro/skills/leftoff/` |
| Lingma | Experimental | `lingma` | `agents/lingma.md` | `~/.lingma/skills/leftoff/` |
| Mistral Vibe | Experimental | `mistral-vibe` | `agents/mistral-vibe.md` | `~/.mistral-vibe/skills/leftoff/` |
| Mux | Experimental | `mux` | `agents/mux.md` | `~/.mux/skills/leftoff/` |
| OpenHands | Experimental | `openhands` | `agents/openhands.md` | `~/.openhands/skills/leftoff/` |
| Qoder | Experimental | `qoder` | `agents/qoder.md` | `~/.qoder/skills/leftoff/` |
| Qwen Code | Experimental | `qwen-code`, `qwen` | `agents/qwen-code.md` | `~/.qwen-code/skills/leftoff/` |
| Rovo Dev | Experimental | `rovo-dev`, `rovo` | `agents/rovo-dev.md` | `~/.rovo/skills/leftoff/` |
| Tabnine CLI | Experimental | `tabnine-cli`, `tabnine` | `agents/tabnine-cli.md` | `~/.tabnine/skills/leftoff/` |
| Trae / Trae CN | Experimental | `trae-trae-cn`, `trae`, `trae-cn` | `agents/trae-trae-cn.md` | `~/.trae/skills/leftoff/` |
| Warp | Experimental | `warp` | `agents/warp.md` | `~/.warp/skills/leftoff/` |
| Windsurf | Experimental | `windsurf` | `agents/windsurf.md` | `~/.windsurf/skills/leftoff/` |
| Zed | Experimental | `zed` | `agents/zed.md` | `~/.zed/skills/leftoff/` |
