#!/usr/bin/env sh
set -eu

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
plugin_root="${PLUGIN_ROOT:-$(CDPATH= cd -- "$script_dir/.." && pwd)}"

version="$(tr -d '[:space:]' < "${plugin_root}/VERSION")"
tag="${TAG:-v${version}}"
tag_version="${tag#v}"

test "$version" = "$tag_version"
jq -e --arg v "$tag_version" '.version == $v' "${plugin_root}/.claude-plugin/plugin.json" >/dev/null
jq -e --arg v "$tag_version" '.version == $v' "${plugin_root}/.codex-plugin/plugin.json" >/dev/null

echo "version ok: ${tag_version}"
