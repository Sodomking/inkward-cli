#!/bin/sh
# INKWARD native bootstrap. Inspect before running. No Node, Git, Go or sudo.
set -eu
version='2.0.0'
yes=0; plan=0; dry=0; no_open=0; exam=''; trial=''
while [ "$#" -gt 0 ]; do
  case "$1" in
    --yes) yes=1;; --plan) plan=1;; --dry-run) dry=1;; --no-open) no_open=1;;
    --exam|--trial) key="$1"; shift; [ "$#" -gt 0 ] || { echo 'Missing value' >&2; exit 1; }; case "$key" in --exam) exam="$1";; --trial) trial="$1";; esac;;
    *) echo 'Usage: sh start.sh --yes [--plan] [--dry-run] [--no-open] [--exam EXAM] [--trial big|small]' >&2; exit 1;;
  esac
  shift
done
case "$exam" in ''|english1|english2|postgraduate|cet4|cet6|ielts|ielts-academic|ielts-general) :;; *) echo 'Invalid exam' >&2; exit 1;; esac
case "$trial" in ''|big|small) :;; *) echo 'Invalid trial' >&2; exit 1;; esac
if [ -n "$trial" ]; then case "$exam" in ''|english1|english2|postgraduate) :;; *) echo 'Trial requires English I or II' >&2; exit 1;; esac; fi
[ "$(uname -s)" = Darwin ] || { echo 'This starter supports macOS. Windows: use start.ps1.' >&2; exit 1; }
arch=$(uname -m)
# Prefer native arm64 even when the agent shell is running under Rosetta.
if [ "$(/usr/sbin/sysctl -n hw.optional.arm64 2>/dev/null || true)" = 1 ]; then arch=arm64; fi
case "$arch" in arm64) arch=arm64;; x86_64) arch=amd64;; *) echo 'Unsupported Mac architecture' >&2; exit 1;; esac
asset="inkward-darwin-$arch"
base="https://github.com/Sodomking/inkward-cli/releases/download/v$version"
if [ "$plan" = 1 ]; then printf '{"version":"%s","os":"darwin","arch":"%s","asset":"%s","url":"%s/%s"}\n' "$version" "$arch" "$asset" "$base" "$asset"; exit 0; fi
if [ "$yes" = 0 ]; then printf 'Configure INKWARD direct access and open the website? [y/N] '; read -r answer; case "$answer" in y|Y|yes|YES) :;; *) exit 1;; esac; fi
umask 077
dest="$HOME/Library/Application Support/INKWARD/bin/$version"
[ ! -L "$dest" ] || { echo 'Install directory must not be a symlink' >&2; exit 1; }
mkdir -p "$dest"
tmp=$(mktemp -d "${TMPDIR:-/tmp}/inkward-download.XXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
fetch() { curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 10 --max-time 120 --retry 2 "$1" -o "$2"; }
fetch "$base/SHA256SUMS" "$tmp/SHA256SUMS"
expected=$(awk -v f="$asset" '$2==f {print $1}' "$tmp/SHA256SUMS")
[ "${#expected}" = 64 ] || { echo 'Missing or invalid checksum' >&2; exit 1; }
case "$expected" in *[!0-9a-f]*) echo 'Invalid checksum' >&2; exit 1;; esac
binary="$dest/inkward"
actual=''
if [ -f "$binary" ] && [ ! -L "$binary" ]; then actual=$(shasum -a 256 "$binary" | awk '{print $1}'); fi
if [ "$actual" != "$expected" ]; then
  fetch "$base/$asset" "$tmp/inkward"
  actual=$(shasum -a 256 "$tmp/inkward" | awk '{print $1}')
  [ "$actual" = "$expected" ] || { echo 'Checksum mismatch; not executed' >&2; exit 1; }
  chmod 700 "$tmp/inkward"
  mv -f "$tmp/inkward" "$binary"
fi
set -- setup --yes --json
[ "$dry" = 0 ] || set -- "$@" --dry-run
[ "$no_open" = 0 ] || set -- "$@" --no-open
[ -z "$exam" ] || set -- "$@" --exam "$exam"
[ -z "$trial" ] || set -- "$@" --trial "$trial"
"$binary" "$@"
