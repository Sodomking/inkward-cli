#!/bin/sh
# Tiny browser entry. No Node, background service, download or proxy mutation.
set -eu
exam=''; trial=''; yes=0; print_url=0
while [ "$#" -gt 0 ]; do
  case "$1" in
    --exam|--trial) key="$1"; shift; [ "$#" -gt 0 ] || { echo 'Missing value' >&2; exit 1; }; case "$key" in --exam) exam="$1";; --trial) trial="$1";; esac ;;
    --yes) yes=1;;
    --print-url) print_url=1;;
    *) echo 'Usage: sh start.sh [--exam english1|postgraduate|cet4|cet6|ielts|ielts-general] [--trial big|small] [--yes] [--print-url]' >&2; exit 1;;
  esac
  shift
done
case "$exam" in english2) exam=postgraduate;; ielts-academic) exam=ielts;; esac
case "$exam" in ''|english1|postgraduate|cet4|cet6|ielts|ielts-general) :;; *) echo 'Invalid exam' >&2; exit 1;; esac
route='/home'
if [ -n "$exam" ]; then route="/exams?exam=$exam"; fi
if [ -n "$trial" ]; then
  case "$trial" in big|small) :;; *) echo 'Invalid trial' >&2; exit 1;; esac
  case "$exam" in
    ''|english1) topic="english1-$trial-2026";;
    postgraduate) if [ "$trial" = big ]; then topic=postgraduate-english2-2026; else topic=postgraduate-english2-small-2026; fi;;
    *) echo 'Trials are available for english1/postgraduate only' >&2; exit 1;;
  esac
  route="/brief?topic=$topic"
fi
url="https://rimbaud.asia/writing/#$route"
printf '%s\n' "$url"
[ "$print_url" = 0 ] || exit 0
if [ "$yes" = 0 ]; then
  [ -t 0 ] || { echo 'Use --yes only when the user requested opening the website.' >&2; exit 1; }
  printf 'Open the website? [y/N] '; read -r answer
  case "$answer" in y|Y|yes|YES) :;; *) exit 0;; esac
fi
case "$(uname -s)" in Darwin) open "$url";; Linux) xdg-open "$url";; *) echo 'Open the URL above in your browser.' >&2; exit 1;; esac
printf '%s\n' 'Browser requested. Login and payment remain in the website.'
