#!/bin/sh
# usage  run.sh <model> <repo|relai> <task> <run-number> <out.jsonl>
# repo: the agent is told to continue; relai: the same plus the output of `relai handoff`. Read, edit, run python and git only.
set -eu
model=$1 cond=$2 task=$3 n=$4 out=$5
here=$(cd "$(dirname "$0")" && pwd)
w=$(mktemp -d /tmp/ws-resume-XXXXXX)
python3 "$here/build.py" "$task" "$w/project"
cd "$w/project"
base="You are continuing a project another session left unfinished. Finish the interrupted work so the project is correct, with the least effort. Hidden acceptance tests will be run afterwards. Do not commit. When done, reply with one line: DONE: <what you did>."
case $cond in
  repo) prompt=$base ;;
  relai) prompt="$base

Here is the output of \`relai handoff\`:

$(${RELAI_BIN:-relai} handoff)" ;;
esac
res=$(claude -p "$prompt" --model "$model" --output-format json --setting-sources project --strict-mcp-config \
  --no-session-persistence \
  --allowedTools "Read,Edit,Write,Glob,Grep,Bash(git:*),Bash(ls:*),Bash(cat:*),Bash(python3:*)" \
  --disallowedTools "WebFetch,WebSearch,Agent,Task,NotebookEdit" 2>/dev/null || echo '{}')
verdict=$(python3 "$here/hidden.py" "$task" "$w/project" 2>&1 | tail -1)
ok=false; [ "$verdict" = PASS ] && ok=true
printf '%s' "$res" | jq -c --arg m "$model" --arg c "$cond" --arg t "$task" --argjson n "$n" --arg v "$verdict" --argjson ok "$ok" \
  '{model:$m,cond:$c,task:$t,run:$n,pass:$ok,verdict:$v,turns:.num_turns,cost:.total_cost_usd,
    input:((.usage.input_tokens//0)+(.usage.cache_read_input_tokens//0)+(.usage.cache_creation_input_tokens//0)),output:(.usage.output_tokens//0)}' >> "$out"
rm -rf "$w"
