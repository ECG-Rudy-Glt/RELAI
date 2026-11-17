#!/bin/sh
# Interrupted-work measure with the Codex CLI. Same tasks, conditions and grading as run.sh (Claude).
# usage  [FIXTURES=<dir>] [CODEX_MODEL=<model>] run-codex.sh <repo|relai> <task> <run-number> <out.jsonl>
# The agent must EDIT files: the sandbox is workspace-write (not read-only), network off. Check the flags with
# `codex exec --help` before the first run; adapt this file (or your PowerShell copy) if they differ.
set -eu
cond=$1 task=$2 n=$3 out=$4
here=$(cd "$(dirname "$0")" && pwd)
fix=${FIXTURES:-$here}
w=$(mktemp -d /tmp/ws-resume-XXXXXX)
python3 "$fix/build.py" "$task" "$w/project" >/dev/null
work="$w/project"
[ -d "$w/project/app" ] && work="$w/project/app"
cd "$work"
base="You are continuing a project another session left unfinished. Finish the interrupted work so the project is correct, with the least effort. Hidden acceptance tests will be run afterwards. Do not commit. When done, reply with one line: DONE: <what you did>."
case $cond in
  repo) prompt=$base ;;
  relai) prompt="$base

Here is the output of \`relai handoff\`:

$(${RELAI_BIN:-relai} handoff)" ;;
esac
model_flag=""
[ -n "${CODEX_MODEL:-}" ] && model_flag="-m $CODEX_MODEL"
# For a multi-repository task the linked repositories are in the parent folder: add it as a writable/readable root
# if your Codex version needs it (`--add-dir ..`), and say so in the report.
# shellcheck disable=SC2086
codex exec --sandbox workspace-write --skip-git-repo-check --ephemeral --json -o "$w/last.txt" $model_flag "$prompt" > "$w/events.jsonl" 2>"$w/stderr.txt" || true
verdict=$(python3 "$fix/hidden.py" "$task" "$w/project" 2>&1 | tail -1)
ok=false; [ "$verdict" = PASS ] && ok=true
input=$(jq -rs '[.[]|select(.type=="turn.completed")|.usage.input_tokens//0]|add//0' "$w/events.jsonl" 2>/dev/null || echo 0)
cached=$(jq -rs '[.[]|select(.type=="turn.completed")|.usage.cached_input_tokens//0]|add//0' "$w/events.jsonl" 2>/dev/null || echo 0)
output=$(jq -rs '[.[]|select(.type=="turn.completed")|.usage.output_tokens//0]|add//0' "$w/events.jsonl" 2>/dev/null || echo 0)
commands=$(jq -rs '[.[]|select(.type=="item.completed" and .item.type=="command_execution")|.item.id]|unique|length' "$w/events.jsonl" 2>/dev/null || echo 0)
jq -nc --arg f "$(basename "$fix")" --arg m "codex${CODEX_MODEL:+:$CODEX_MODEL}" --arg c "$cond" --arg t "$task" --argjson n "$n" \
  --arg v "$verdict" --argjson ok "$ok" --argjson i "$input" --argjson ca "$cached" --argjson o "$output" --argjson k "$commands" \
  '{fixtures:$f,model:$m,cond:$c,task:$t,run:$n,pass:$ok,verdict:$v,commands:$k,input:$i,cached_input:$ca,output:$o}' >> "$out"
rm -rf "$w"
