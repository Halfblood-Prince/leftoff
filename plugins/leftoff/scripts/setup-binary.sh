#!/usr/bin/env sh
set -eu

repo="Halfblood-Prince/leftoff"
version=""
assume_yes="0"

usage() {
  cat <<'USAGE'
Usage: scripts/setup-binary.sh [--version v1.0.0] [--repo OWNER/REPO] [--yes]

Downloads the platform release bundle with gh, verifies the GitHub artifact
attestation and SHA256SUMS entry, then installs the binary under:

  bin/.leftoff/<os>_<arch>/

Network access is explicit. Without --yes, the script asks before downloading.
USAGE
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --version)
      version="${2:?missing --version value}"
      shift 2
      ;;
    --repo)
      repo="${2:?missing --repo value}"
      shift 2
      ;;
    --yes)
      assume_yes="1"
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

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
plugin_root="$(CDPATH= cd -- "$script_dir/.." && pwd)"

if [ -z "$version" ]; then
  if [ -f "$plugin_root/VERSION" ]; then
    version="v$(tr -d '[:space:]' < "$plugin_root/VERSION")"
  else
    echo "missing --version and VERSION file" >&2
    exit 1
  fi
fi

case "$version" in
  v*) tag="$version" ;;
  *) tag="v$version" ;;
esac

command -v gh >/dev/null 2>&1 || {
  echo "gh is required to download and verify release provenance" >&2
  exit 1
}
command -v unzip >/dev/null 2>&1 || {
  echo "unzip is required to extract the release bundle" >&2
  exit 1
}

os="$(uname -s 2>/dev/null | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m 2>/dev/null | tr '[:upper:]' '[:lower:]')"

case "$os" in
  linux*) goos="linux" ;;
  darwin*) goos="darwin" ;;
  msys*|mingw*|cygwin*) goos="windows" ;;
  *) echo "unsupported OS: $os" >&2; exit 1 ;;
esac

case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *) echo "unsupported architecture: $arch" >&2; exit 1 ;;
esac

asset="leftoff_${tag}_${goos}_${goarch}.zip"

if [ "$assume_yes" != "1" ]; then
  cat <<EOF
This will download from GitHub:
  repo:  $repo
  tag:   $tag
  asset: $asset

The script will verify GitHub artifact provenance and SHA256SUMS before
installing the binary under this plugin root:
  $plugin_root/bin/.leftoff/${goos}_${goarch}/
EOF
  printf 'Continue? [y/N] '
  read answer
  case "$answer" in
    y|Y|yes|YES) ;;
    *) echo "cancelled"; exit 1 ;;
  esac
fi

tmp="${TMPDIR:-/tmp}/leftoff-setup.$$"
mkdir -p "$tmp"
cleanup() {
  rm -rf "$tmp"
}
trap cleanup EXIT INT TERM

gh release download "$tag" --repo "$repo" --pattern "$asset" --pattern SHA256SUMS --dir "$tmp"
gh attestation verify "$tmp/$asset" --repo "$repo"

expected="$(awk -v asset="$asset" '$2 == asset { print $1 }' "$tmp/SHA256SUMS")"
if [ -z "$expected" ]; then
  echo "missing SHA256SUMS entry for $asset" >&2
  exit 1
fi

if command -v sha256sum >/dev/null 2>&1; then
  actual="$(sha256sum "$tmp/$asset" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  actual="$(shasum -a 256 "$tmp/$asset" | awk '{ print $1 }')"
else
  echo "sha256sum or shasum is required for checksum verification" >&2
  exit 1
fi

if [ "$expected" != "$actual" ]; then
  echo "checksum mismatch for $asset" >&2
  exit 1
fi

mkdir -p "$tmp/extract"
unzip -q "$tmp/$asset" -d "$tmp/extract"

exe="leftoff"
if [ "$goos" = "windows" ]; then
  exe="leftoff.exe"
fi

source_bin="$tmp/extract/leftoff_${tag}_${goos}_${goarch}/bin/$exe"
if [ ! -f "$source_bin" ]; then
  echo "release bundle did not contain bin/$exe" >&2
  exit 1
fi

install_dir="$plugin_root/bin/.leftoff/${goos}_${goarch}"
mkdir -p "$install_dir"
cp "$source_bin" "$install_dir/$exe"
chmod +x "$install_dir/$exe"

echo "installed verified leftoff binary: $install_dir/$exe"
echo "launcher: $plugin_root/bin/leftoff"
