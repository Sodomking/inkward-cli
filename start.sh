#!/bin/sh
# INKWARD native bootstrap. Inspect before running. No Node, Git, Go or sudo.
set -eu
version='2.0.1'
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
if [ "$yes" = 0 ]; then printf 'Set up INKWARD and open the website (company services included on first use)? [y/N] '; read -r answer; case "$answer" in y|Y|yes|YES) :;; *) exit 1;; esac; fi
umask 077
dest="$HOME/Library/Application Support/INKWARD/bin/$version"
[ ! -L "$dest" ] || { echo 'Install directory must not be a symlink' >&2; exit 1; }
mkdir -p "$dest"
tmp=$(mktemp -d "${TMPDIR:-/tmp}/inkward-download.XXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM
fetch() { curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 10 --max-time 120 --retry 1 "$1" -o "$2"; }
mirror='https://static.rimbaud.asia/releases/7e2d2c0ee1641998/cli/v2.0.1'
api_asset=''; api_sums=''; api_loaded=0
load_api() {
  [ "$api_loaded" = 0 ] || return 0
  api_loaded=1
  if fetch "https://api.github.com/repos/Sodomking/inkward-cli/releases/tags/v$version" "$tmp/release.json"; then
    i=0
    while [ "$i" -lt 20 ]; do
      name=$(/usr/bin/plutil -extract "assets.$i.name" raw -o - "$tmp/release.json" 2>/dev/null) || break
      if [ "$name" = "$asset" ] || [ "$name" = SHA256SUMS ]; then
        endpoint=$(/usr/bin/plutil -extract "assets.$i.url" raw -o - "$tmp/release.json")
        id=${endpoint#https://api.github.com/repos/Sodomking/inkward-cli/releases/assets/}
        case "$id" in ''|*[!0-9]*) echo 'Invalid official asset endpoint' >&2; exit 1;; esac
        if [ "$name" = "$asset" ]; then api_asset="$endpoint"; else api_sums="$endpoint"; fi
      fi
      i=$((i + 1))
    done
  fi
}
fetch_asset() {
  wanted="$1"; target="$2"
  # Only this site's request bypasses terminal proxy variables. System VPN/TUN may still apply.
  if curl --noproxy '.rimbaud.asia' --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 8 --max-time 90 --retry 1 "$mirror/$wanted" -o "$target"; then return 0; fi
  load_api
  if [ "$wanted" = SHA256SUMS ]; then endpoint="$api_sums"; else endpoint="$api_asset"; fi
  if [ -n "$endpoint" ]; then
    if curl --fail --silent --show-error --location --proto '=https' --proto-redir '=https' --connect-timeout 10 --max-time 120 --retry 1 -H 'Accept: application/octet-stream' "$endpoint" -o "$target"; then return 0; fi
  fi
  fetch "$base/$wanted" "$target"
}
fetch_asset SHA256SUMS "$tmp/SHA256SUMS"
[ "$(shasum -a 256 "$tmp/SHA256SUMS" | awk '{print $1}')" = '7e2d2c0ee16419982b229755d74339bb36d22dc81cdaac5558376e6f772e527c' ] || { echo 'Release manifest mismatch' >&2; exit 1; }
expected=$(awk -v f="$asset" '$2==f {print $1}' "$tmp/SHA256SUMS")
[ "${#expected}" = 64 ] || { echo 'Missing or invalid checksum' >&2; exit 1; }
case "$expected" in *[!0-9a-f]*) echo 'Invalid checksum' >&2; exit 1;; esac
binary="$dest/inkward"
actual=''
if [ -f "$binary" ] && [ ! -L "$binary" ]; then actual=$(shasum -a 256 "$binary" | awk '{print $1}'); fi
if [ "$actual" != "$expected" ]; then
  fetch_asset "$asset" "$tmp/inkward"
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
