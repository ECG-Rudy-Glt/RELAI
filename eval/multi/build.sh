#!/bin/sh
# Build the multi-repository fixture in <dir>: four small repositories side by side.
#   web-app      the project the agent will work on; consumes billing-api; its README and TODO are stale
#   billing-api  moved after web-app's last commit: /v1/invoices replaced by /v2/invoices
#   auth-svc     linked, did not change anything that matters
#   notes-tool   unrelated, busy
# Right answer for web-app: wa-3, the invoices page, against /v2/invoices (not /v1).
set -eu
dir=$1
variant=${2:-easy}   # easy: billing-api's README says /v2. hard: its README is stale too (still /v1); the change is only in a commit and a closed task, and six busy decoys sit next to it.
mkdir -p "$dir"
cd "$dir"
commit() { # repo date message
  GIT_AUTHOR_NAME=eval GIT_AUTHOR_EMAIL=e@e GIT_COMMITTER_NAME=eval GIT_COMMITTER_EMAIL=e@e \
  GIT_AUTHOR_DATE="$2T12:00:00Z" GIT_COMMITTER_DATE="$2T12:00:00Z" git -C "$1" commit -q -m "$3"
}
new() { mkdir -p "$1/.beads" && git -C "$1" init -q -b main 2>/dev/null || { mkdir -p "$1"; git -C "$1" init -q -b main; }; }
tasks() { # repo lines...
  r=$1; shift; printf '%s\n' "$@" > "$r/.beads/issues.jsonl"
}

new web-app
cat > web-app/README.md <<'R'
# web-app
Customer dashboard. Talks to billing-api (invoices) and auth-svc (login).
Invoices: call `GET /v1/invoices` of billing-api.
R
cat > web-app/TODO.md <<'R'
- [x] login page
- [ ] invoices page (GET /v1/invoices)
- [ ] dark mode
R
printf 'id: web-app\nsummary: Customer dashboard.\nlinks: [billing-api, auth-svc]\n' > web-app/relai.yaml
tasks web-app \
 '{"id":"wa-3","title":"Invoices page","status":"open","priority":1,"issue_type":"task"}' \
 '{"id":"wa-4","title":"Dark mode","status":"open","priority":3,"issue_type":"task"}' \
 '{"id":"wa-2","title":"Login page","status":"closed","priority":1,"issue_type":"task"}'
git -C web-app add -A; commit web-app 2026-11-01 "feat: login page"

new billing-api
cat > billing-api/README.md <<'R'
# billing-api
Invoices service. `GET /v1/invoices` returns all invoices.
R
printf 'id: billing-api\nsummary: Invoices service.\nprovides: [invoices.list]\n' > billing-api/relai.yaml
tasks billing-api '{"id":"ba-1","title":"Cursor pagination","status":"closed","priority":1,"issue_type":"task"}'
git -C billing-api add -A; commit billing-api 2026-10-20 "feat: GET /v1/invoices"
if [ "$variant" = easy ]; then
cat > billing-api/README.md <<'R'
# billing-api
Invoices service. `GET /v2/invoices?cursor=` returns a page of invoices. The /v1 endpoint was removed.
R
else
tasks billing-api '{"id":"ba-1","title":"Cursor pagination","status":"closed","priority":1,"issue_type":"task"}' '{"id":"ba-7","title":"Replace /v1/invoices by /v2/invoices (cursor pagination), remove /v1","status":"closed","priority":1,"issue_type":"task"}'
echo "// handler for /v2/invoices" > billing-api/handler.go
fi
git -C billing-api add -A; commit billing-api 2026-11-10 "feat!: replace /v1/invoices by /v2/invoices with cursor pagination, /v1 removed"

new auth-svc
printf 'id: auth-svc\nsummary: Login service.\nprovides: [auth.login]\n' > auth-svc/relai.yaml
tasks auth-svc '{"id":"au-1","title":"Rotate keys","status":"open","priority":3,"issue_type":"task"}'
echo "# auth-svc" > auth-svc/README.md
git -C auth-svc add -A; commit auth-svc 2026-10-15 "feat: login"

new notes-tool
printf 'id: notes-tool\nsummary: Personal notes.\n' > notes-tool/relai.yaml
tasks notes-tool '{"id":"nt-1","title":"Export to markdown","status":"in_progress","priority":1,"issue_type":"task"}' '{"id":"nt-2","title":"Tags","status":"open","priority":2,"issue_type":"task"}'
echo "# notes-tool" > notes-tool/README.md
git -C notes-tool add -A; commit notes-tool 2026-11-12 "feat: notes list"

if [ "$variant" = hard ]; then
  for i in 1 2 3 4 5 6; do
    new tool-$i
    printf 'id: tool-%s\nsummary: Side project %s.\n' $i $i > tool-$i/relai.yaml
    tasks tool-$i "{\"id\":\"t$i-1\",\"title\":\"Move the v2 parser to the new API\",\"status\":\"open\",\"priority\":1,\"issue_type\":\"task\"}"
    echo "# tool-$i, now with a v2 parser" > tool-$i/README.md
    git -C tool-$i add -A; commit tool-$i 2026-11-1$((i+1)) "feat: v2 parser"
  done
fi
