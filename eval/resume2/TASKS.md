# resume2 tasks

**s1 (logsum).** The session left a log-level counter whose reader was already rewritten to stream lines; the next step is `top_messages(path, n)` in `logsum/levels.py`. Wrong continuation: `Path.read_text().splitlines()` plus a Counter. Decisive fact: the commit message "stream the log line by line (prod logs are several GB...)" and the `### Pitfalls` line of `.relai/notes.md`. Hidden check wraps `open` and fails on any `read`/`readlines`.

**s2 (todo).** A JSON todo store with `add`, `list_items`, `done`; the next step is `delete(path, item_id)`. Wrong continuation: remove the item from the list. Decisive fact: `### Decisions` in `.relai/notes.md` (an external sync merges by id, so deletion must set `deleted_at`). Hidden check needs the item still in the file with a truthy `deleted_at` and hidden from `list_items`.

**s3 (dash).** A state store that already writes atomically; the next step is a new `dash/export.py` with `export_csv`. Wrong continuation: `open(path, "w")` with csv.writer. Decisive fact: the fix commit message and the `### Pitfalls` line (the dashboard polls out/; write a temp file then `os.replace`). Hidden check fails if the final path is ever opened for writing, or if temp files remain.

**m1 (app, store, tools).** app has `load_record` reading `<root>/<id>.json`; next step `title_of(root, rid)`. Wrong continuation: reuse `load_record` unchanged. Decisive fact: store commit of 2026-11-05 (after app's last commit, 2026-11-01) and its notes line: records are now sharded as `<root>/<id[:2]>/<id>.json`. `tools` is a distractor. Hidden check builds only the sharded layout.

**m2 (app, platform, ui).** app has `get_port` reading `PORT`; next step `get_workers(env)` default 1. Wrong continuation: `env.get("WORKERS")`. Decisive fact: platform commit of 2026-11-04 and notes: services only receive `SVC_`-prefixed variables. `ui` is a distractor. Hidden check requires `SVC_WORKERS` to be honored.

**m3 (app, shared-dates, ui).** app has `week_of` (Sunday-start); next step `weekly_totals(rows)`. Wrong continuation: group with the existing `week_of`. Decisive fact: shared-dates commit of 2026-11-05 and notes: weeks start on Monday everywhere. `ui` is a distractor. Hidden check expects Monday-start ISO keys.
