#!/bin/sh
# Multi-repository measure: usage  run.sh <model> <condition: none|docs|relai> <run-number> <out.jsonl>
# Needs claude (logged in), git, jq and a relai binary (RELAI_BIN, default: relai on PATH).
set -eu
model=$1 cond=$2 n=$3 out=$4
here=$(cd "$(dirname "$0")" && pwd)
w=$(mktemp -d /tmp/ws-multi-XXXXXX)   # neutral path: nothing says which fixture or condition
sh "$here/build.sh" "$w/projects" >/dev/null
cd "$w/projects"
base="The folder $w/projects holds several git projects. You will work on web-app today. Find where its work stands and choose the single next step. Change nothing. End your answer with one line: NEXT: <task id> — <the step>."
case $cond in
  none) prompt=$base ;;
  docs) prompt="$base Start by reading the documentation of the projects (README, TODO files)." ;;
  relai) prompt="$base

Here is the output of \`relai handoff --workspace .\`:

$(${RELAI_BIN:-relai} handoff --workspace .)" ;;
esac
res=$(claude -p "$prompt" --model "$model" --output-format json --setting-sources project --strict-mcp-config \
  --no-session-persistence \
  --allowedTools "Read,Glob,Grep,Bash(git:*),Bash(ls:*),Bash(cat:*),Bash(head:*),Bash(find:*),Bash(wc:*)" \
  --disallowedTools "Edit,Write,NotebookEdit,WebFetch,WebSearch,Agent,Task" 2>/dev/null || echo '{}')
next=$(printf '%s' "$res" | jq -r '.result // ""' | grep -E '^NEXT:' | tail -1 || true)
ok=false
# right: the step names the v2 endpoint; wrong (trap): wires /v1, or another task
if printf '%s' "$next" | grep -qiE 'wa-3|invoice' && printf '%s' "$next" | grep -qiE 'v2|cursor'; then ok=true; fi
printf '%s' "$res" | jq -c --arg m "$model" --arg c "$cond" --argjson n "$n" --arg next "$next" --argjson ok "$ok" \
  '{model:$m,cond:$c,run:$n,correct:$ok,next:$next,turns:.num_turns,cost:.total_cost_usd,
    input:((.usage.input_tokens//0)+(.usage.cache_read_input_tokens//0)+(.usage.cache_creation_input_tokens//0)),output:(.usage.output_tokens//0)}' >> "$out"
rm -rf "$w"
