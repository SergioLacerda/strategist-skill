#!/usr/bin/env bash
# Strategist · Spell Charge — progress bar loader with token meter (pure ANSI)
# Usage: ./spell-charge.sh

AMBER=$'\e[38;5;179m'
EMBER=$'\e[38;5;173m'
TRACK=$'\e[38;5;94m'
GREEN=$'\e[38;5;114m'
PHOS=$'\e[38;5;114m'
DIM=$'\e[38;5;94m'
RESET=$'\e[0m'
HIDE=$'\e[?25l'; SHOW=$'\e[?25h'

width=28

cleanup() { printf '%s\n' "$SHOW"; }
trap cleanup EXIT INT

printf '%s%s$ strategist compile --spell%s\n' "$HIDE" "$DIM" "$RESET"

for (( p=0; p<=width; p++ )); do
  filled=""; head=""; rest=""
  for (( i=0; i<width; i++ )); do
    if   (( i < p ));  then filled+="█"
    elif (( i == p )); then head="▓"
    else                    rest+="░"
    fi
  done
  pct=$(( p * 100 / width ))

  # meter: token counter grows with progress
  tok=$(( p * 4400 ))
  if (( tok >= 1000 )); then
    tokfmt=$(awk "BEGIN{printf \"%.1fk\", $tok/1000}")
  else
    tokfmt="$tok"
  fi

  printf '\r  %s✶ channeling mana%s  %s%s%s%s%s%s%s %s%3d%%%s %s·%s %s↑ %s%s %stokens%s' \
    "$AMBER" "$RESET" \
    "$AMBER" "$filled" "$EMBER" "$head" "$TRACK" "$rest" "$RESET" \
    "$AMBER" "$pct" "$RESET" \
    "$DIM" "$RESET" \
    "$PHOS" "$tokfmt" "$RESET" \
    "$DIM" "$RESET"
  sleep 0.06
done

printf ' %s✓%s\n' "$GREEN" "$RESET"
