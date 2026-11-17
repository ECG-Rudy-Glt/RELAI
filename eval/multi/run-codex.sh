#!/bin/sh
# Same measure as run.sh, with the Codex CLI. UNTESTED: written without Codex installed. Check the flags
# against `codex exec --help` first and adapt this file if they differ.
# usage  [VARIANT=easy|hard] [CODEX_MODEL=<model>] run-codex.sh <none|docs|relai> <run-number> <out.jsonl>
# Needs codex (logged in), git, jq and a relai binary (RELAI_BIN, default: relai on PATH).
set -eu
cond=$1 n=$2 out=$3
here=$(cd "$(dirname "$0")" && pwd)
w=$(mktemp -d /tmp/ws-multi-XXXXXX)
sh "$here/build.sh" "$w/projects" "${VARIANT:-easy}" >/dev/null
cd "$w/projects"
base="The folder $w/projects holds several git projects. You will work on web-app today. Find where its work stands and choose the single next step. Change nothing. End your answer with one line: NEXT: <task id> — <the step>."
case $cond in
  none) prompt=$base ;;
  docs) prompt="$base Start by reading the documentation of the projects (README, TODO files)." ;;
  relai) prompt="$base

Here is the output of \`relai handoff --workspace .\`:

$(${RELAI_BIN:-relai} handoff --workspace .)" ;;
esac
model_flag=""
[ -n "${CODEX_MODEL:-}" ] && model_flag="-m $CODEX_MODEL"
# read-only sandbox: the agent can read files and run read-only commands, nothing is written
# shellcheck disable=SC2086
codex exec --sandbox read-only --skip-git-repo-check --json -o "$w/last.txt" $model_flag "$prompt" > "$w/events.jsonl" 2>/dev/null || true
next=$(grep -E '^NEXT:' "$w/last.txt" 2>/dev/null | tail -1 || true)
ok=false
if printf '%s' "$next" | grep -qiE 'wa-3|invoice' && printf '%s' "$next" | grep -qiE 'v2|cursor'; then ok=true; fi
# tokens: the last event that carries a usage object (field names may differ between versions)
input=$(jq -rs '[.[]|.. |objects|select(has("input_tokens"))]|last|.input_tokens // 0' "$w/events.jsonl" 2>/dev/null || echo 0)
output=$(jq -rs '[.[]|.. |objects|select(has("output_tokens"))]|last|.output_tokens // 0' "$w/events.jsonl" 2>/dev/null || echo 0)
turns=$(grep -c '"command_execution"' "$w/events.jsonl" 2>/dev/null || echo 0)
jq -nc --arg v "${VARIANT:-easy}" --arg m "codex${CODEX_MODEL:+:$CODEX_MODEL}" --arg c "$cond" --argjson n "$n" --arg next "$next" \
  --argjson ok "$ok" --argjson i "${input:-0}" --argjson o "${output:-0}" --argjson t "$turns" \
  '{variant:$v,model:$m,cond:$c,run:$n,correct:$ok,next:$next,commands:$t,input:$i,output:$o}' >> "$out"
rm -rf "$w"
