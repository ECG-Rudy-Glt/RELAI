# Helpers shared by the fixture build scripts. Sourced, not executed.
#
# A fixture is a small, scripted project whose correct next step is known in advance.
# Each build.sh writes the repo into the directory given as $1, with fixed author, dates
# and task IDs so that every run starts from the same state.
set -euo pipefail

fx_init() {
	FX_DIR=$1
	mkdir -p "$FX_DIR"
	cd "$FX_DIR"
	git init -q -b main
	git config user.name "Sam Doe"
	git config user.email "sam@example.com"
	git config commit.gpgsign false
	FX_DAY=0
}

# commit "message" — stage everything in the fixture and commit it one day after the previous one.
commit() {
	FX_DAY=$((FX_DAY + 1))
	local when
	when=$(date -u -d "2026-06-01 +${FX_DAY} days 10:00" +%Y-%m-%dT%H:%M:%SZ)
	git add -A
	GIT_AUTHOR_DATE=$when GIT_COMMITTER_DATE=$when git commit -q -m "$1"
}

# write path <<'EOF' … EOF — create a file and its parent directories.
write() {
	mkdir -p "$(dirname "$1")"
	cat >"$1"
}

# bd_init prefix — Beads task graph, without the AGENTS.md and git hooks that bd adds by default.
bd_init() {
	bd init --non-interactive --skip-agents --skip-hooks -p "$1" >/dev/null 2>&1
	# bd commits on its own; fold that commit into the next fixture commit.
	if git log -1 --format=%s | grep -q '^bd init'; then
		git reset -q --soft HEAD~1
	fi
}

# task id priority "title" [extra bd create flags…]
task() {
	local id=$1 prio=$2 title=$3
	shift 3
	bd create "$title" --id "$id" -p "$prio" -q "$@" >/dev/null
}

done_task() { bd close "$1" --reason "${2:-done}" -q >/dev/null; }
start_task() { bd update "$1" --status in_progress -q >/dev/null; }
