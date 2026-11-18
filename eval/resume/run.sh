#!/bin/sh
# usage  [FIXTURES=<dir with build.py and hidden.py>] run.sh <model> <repo|relai> <task> <run-number> <out.jsonl>
# A task whose build makes an `app` folder is multi-repository: the agent works in app, its linked repositories sit next to it.
# repo: the agent is told to continue; relai: the same plus the output of `relai handoff`. Read, edit, run python and git only.
set -eu
model=$1 cond=$2 task=$3 n=$4 out=$5
here=$(cd "$(dirname "$0")" && pwd)
w=$(mktemp -d /tmp/ws-resume-XXXXXX)
fix=${FIXTURES:-$here}
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
adddir=""
[ "${ADD_PARENT:-0}" = 1 ] && [ "$work" != "$w/project" ] && adddir="--add-dir $w/project"
# shellcheck disable=SC2086
res=$(claude -p "$prompt" $adddir --model "$model" --output-format json --setting-sources project --strict-mcp-config \
  --no-session-persistence \
  --allowedTools "Read,Edit,Write,Glob,Grep,Bash(git:*),Bash(ls:*),Bash(cat:*),Bash(python3:*)" \
  --disallowedTools "WebFetch,WebSearch,Agent,Task,NotebookEdit" 2>/dev/null || echo '{}')
verdict=$(python3 "$fix/hidden.py" "$task" "$w/project" 2>&1 | tail -1)
ok=false; [ "$verdict" = PASS ] && ok=true
printf '%s' "$res" | jq -c --arg ap "${ADD_PARENT:-0}" --arg f "$(basename "$fix")" --arg m "$model" --arg c "$cond" --arg t "$task" --argjson n "$n" --arg v "$verdict" --argjson ok "$ok" \
  '{add_parent:$ap,fixtures:$f,model:$m,cond:$c,task:$t,run:$n,pass:$ok,verdict:$v,turns:.num_turns,cost:.total_cost_usd,
    input:((.usage.input_tokens//0)+(.usage.cache_read_input_tokens//0)+(.usage.cache_creation_input_tokens//0)),output:(.usage.output_tokens//0)}' >> "$out"
rm -rf "$w"
