#!/usr/bin/env sh
set -eu

target=""
agent="generic"
mode="copy"
dry_run="0"

usage() {
  cat <<'USAGE'
Usage: ./install.sh [--agent NAME] [--target PATH] [--mode copy|symlink] [--dry-run]

Installs this repository as a leftoff skill.

Known agent targets are listed in agents/supported.md.

Default target:
  ~/.leftoff/skills/leftoff

An explicit --target overrides the agent default. The script does not overwrite
an existing target.
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
    --mode)
      mode="${2:?missing --mode value}"
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

case "$mode" in
  copy|symlink) ;;
  *)
    echo "unsupported install mode: $mode" >&2
    exit 1
    ;;
esac

source_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

if [ -z "$target" ]; then
  target="$(default_target_for_agent "$agent")"
fi

echo "Agent target: $agent"
echo "Install target: $target"
echo "Install mode: $mode"
echo "Source: $source_dir"

if [ "$dry_run" = "1" ]; then
  echo "Dry run: no files will be changed."
  exit 0
fi

mkdir -p "$(dirname "$target")"

if [ -e "$target" ]; then
  echo "install target already exists: $target" >&2
  exit 1
fi

if [ "$mode" = "symlink" ]; then
  ln -s "$source_dir" "$target"
else
  mkdir -p "$target"
  tar -cf - \
    --exclude '.git' \
    --exclude '.tmp' \
    --exclude './leftoff' \
    --exclude './leftoff.exe' \
    -C "$source_dir" . | tar -xf - -C "$target"
fi

echo "installed leftoff skill to $target"
