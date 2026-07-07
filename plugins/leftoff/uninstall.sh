#!/usr/bin/env sh
set -eu

target=""
agent="generic"
dry_run="0"

usage() {
  cat <<'USAGE'
Usage: ./uninstall.sh [--agent NAME] [--target PATH] [--dry-run]

Removes an installed leftoff skill directory or symlink. This does not delete
the user data store at ~/.leftoff.

Known agent targets are listed in agents/supported.md.
USAGE
}

default_target_for_agent() {
  case "$1" in
    generic)
      printf '%s\n' "${HOME}/.leftoff/skills/leftoff"
      ;;
    claude-code|claude)
      printf '%s\n' "${HOME}/.claude/skills/leftoff"
      ;;
    codex)
      printf '%s\n' "${CODEX_HOME:-${HOME}/.codex}/skills/leftoff"
      ;;
    cursor)
      printf '%s\n' "${HOME}/.cursor/skills/leftoff"
      ;;
    pi)
      printf '%s\n' "${HOME}/.pi/skills/leftoff"
      ;;
    github-copilot-cli|copilot-cli|copilot)
      printf '%s\n' "${HOME}/.github-copilot-cli/skills/leftoff"
      ;;
    opencode)
      printf '%s\n' "${HOME}/.opencode/skills/leftoff"
      ;;
    gemini-cli-antigravity|gemini|google-gemini|antigravity)
      printf '%s\n' "${HOME}/.gemini/skills/leftoff"
      ;;
    factory-ai-droid|factory-droid|droid)
      printf '%s\n' "${HOME}/.factory-ai-droid/skills/leftoff"
      ;;
    openclaw)
      printf '%s\n' "${HOME}/.openclaw/skills/leftoff"
      ;;
    hermes-agent|hermes)
      printf '%s\n' "${HOME}/.hermes/skills/leftoff"
      ;;
    astrbot)
      printf '%s\n' "${HOME}/.astrbot/skills/leftoff"
      ;;
    nanoclaw)
      printf '%s\n' "${HOME}/.nanoclaw/skills/leftoff"
      ;;
    shelley)
      printf '%s\n' "${HOME}/.shelley/skills/leftoff"
      ;;
    auggie-augment|auggie|augment)
      printf '%s\n' "${HOME}/.augment/skills/leftoff"
      ;;
    cline-roo-code|cline|roo|roo-code)
      printf '%s\n' "${HOME}/.cline/skills/leftoff"
      ;;
    codebuddy)
      printf '%s\n' "${HOME}/.codebuddy/skills/leftoff"
      ;;
    continue)
      printf '%s\n' "${HOME}/.continue/skills/leftoff"
      ;;
    crush)
      printf '%s\n' "${HOME}/.crush/skills/leftoff"
      ;;
    deep-agents|deepagents)
      printf '%s\n' "${HOME}/.deep-agents/skills/leftoff"
      ;;
    firebender)
      printf '%s\n' "${HOME}/.firebender/skills/leftoff"
      ;;
    forgecode)
      printf '%s\n' "${HOME}/.forgecode/skills/leftoff"
      ;;
    goose)
      printf '%s\n' "${HOME}/.goose/skills/leftoff"
      ;;
    junie)
      printf '%s\n' "${HOME}/.junie/skills/leftoff"
      ;;
    kilo-code|kilocode)
      printf '%s\n' "${HOME}/.kilo-code/skills/leftoff"
      ;;
    kimi-code-cli|kimi)
      printf '%s\n' "${HOME}/.kimi-code-cli/skills/leftoff"
      ;;
    kiro-cli|kiro)
      printf '%s\n' "${HOME}/.kiro/skills/leftoff"
      ;;
    lingma)
      printf '%s\n' "${HOME}/.lingma/skills/leftoff"
      ;;
    mistral-vibe)
      printf '%s\n' "${HOME}/.mistral-vibe/skills/leftoff"
      ;;
    mux)
      printf '%s\n' "${HOME}/.mux/skills/leftoff"
      ;;
    openhands)
      printf '%s\n' "${HOME}/.openhands/skills/leftoff"
      ;;
    qoder)
      printf '%s\n' "${HOME}/.qoder/skills/leftoff"
      ;;
    qwen-code|qwen)
      printf '%s\n' "${HOME}/.qwen-code/skills/leftoff"
      ;;
    rovo-dev|rovo)
      printf '%s\n' "${HOME}/.rovo/skills/leftoff"
      ;;
    tabnine-cli|tabnine)
      printf '%s\n' "${HOME}/.tabnine/skills/leftoff"
      ;;
    trae-trae-cn|trae|trae-cn)
      printf '%s\n' "${HOME}/.trae/skills/leftoff"
      ;;
    warp)
      printf '%s\n' "${HOME}/.warp/skills/leftoff"
      ;;
    windsurf)
      printf '%s\n' "${HOME}/.windsurf/skills/leftoff"
      ;;
    zed)
      printf '%s\n' "${HOME}/.zed/skills/leftoff"
      ;;
    *)
      echo "unsupported agent: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --agent)
      agent="${2:?missing --agent value}"
      shift 2
      ;;
    --target)
      target="${2:?missing --target value}"
      shift 2
      ;;
    --dry-run)
      dry_run="1"
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [ -z "$target" ]; then
  target="$(default_target_for_agent "$agent")"
fi

echo "Agent target: $agent"
echo "Uninstall target: $target"
echo "Data store retained: ${HOME}/.leftoff"

if [ "$dry_run" = "1" ]; then
  echo "Dry run: no files will be changed."
  exit 0
fi

if [ ! -e "$target" ]; then
  echo "nothing installed at $target"
  exit 0
fi

printf '%s\n' "This will remove the installed skill at: $target"
printf '%s' "Type 'remove leftoff' to continue: "
read answer

if [ "$answer" != "remove leftoff" ]; then
  echo "aborted"
  exit 1
fi

rm -rf "$target"
echo "removed $target"
