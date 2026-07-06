#!/usr/bin/env sh
set -eu

version="$(tr -d '[:space:]' < VERSION)"
tag="${TAG:-v${version}}"
tag_version="${tag#v}"
plugin_root="${PLUGIN_ROOT:-plugins/leftoff}"

if [ ! -f "${plugin_root}/.claude-plugin/plugin.json" ] || [ ! -f "${plugin_root}/.codex-plugin/plugin.json" ]; then
  plugin_root="."
fi

test "$version" = "$tag_version"
jq -e --arg v "$tag_version" '.version == $v' "${plugin_root}/.claude-plugin/plugin.json" >/dev/null
jq -e --arg v "$tag_version" '.version == $v' "${plugin_root}/.codex-plugin/plugin.json" >/dev/null

echo "version ok: ${tag_version}"
