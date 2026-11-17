#!/usr/bin/env python3
"""Usage: build.py <task> <dir>. Builds the state a previous session left behind."""
import os
import subprocess
import sys
import textwrap


def sh(d, *a, date=None):
    env = dict(os.environ, GIT_AUTHOR_NAME="e", GIT_AUTHOR_EMAIL="e@e",
               GIT_COMMITTER_NAME="e", GIT_COMMITTER_EMAIL="e@e")
    if date:
        env["GIT_AUTHOR_DATE"] = env["GIT_COMMITTER_DATE"] = date
    subprocess.run(["git", "-C", d, *a], check=True, env=env, stdout=subprocess.DEVNULL)


def write(d, files):
    for p, c in files.items():
        full = os.path.join(d, p)
        os.makedirs(os.path.dirname(full), exist_ok=True)
        with open(full, "w") as f:
            f.write(textwrap.dedent(c).lstrip("\n"))


def commit(d, date, msg, files):
    write(d, files)
    sh(d, "add", "-A")
    sh(d, "commit", "-q", "-m", msg, date=date + "T10:00:00+0000")


def init(d):
    os.makedirs(d, exist_ok=True)
    sh(d, "init", "-q", "-b", "main")


def notes(decisions, pitfalls, nxt):
    def sec(t, lines):
        return "### %s\n" % t + "".join("- %s\n" % l for l in lines)
    return "\n".join([sec("Decisions", decisions), sec("Pitfalls", pitfalls),
                      sec("Next session", nxt)])


AGENTS = """
Python stdlib only.
Run tests: python3 -m unittest discover -s tests
"""


def meta(task, links=None, summary="Small Python project."):
    y = "id: %s\nsummary: %s\n" % (task, summary)
    if links:
        y += "links: [%s]\n" % ", ".join(links)
    return y


# ---------------------------------------------------------------- single repos

def single(task, d, scaffold, fix, note_files, fix_msg, fix_date="2026-11-06"):
    init(d)
    base = {"AGENTS.md": AGENTS, "relai.yaml": meta(task)}
    base.update(scaffold)
    commit(d, "2026-11-03", "feat: first version", base)
    if fix:
        commit(d, fix_date, fix_msg, fix)
    commit(d, "2026-11-10", "docs: session notes", note_files)


def s1(d):
    scaffold = {
        "logsum/__init__.py": "",
        "logsum/reader.py": """
            from pathlib import Path


            def read_all(path):
                return Path(path).read_text().splitlines()
        """,
        "logsum/levels.py": """
            from .reader import read_all


            def level_of(line):
                parts = line.split(" ", 2)
                return parts[1] if len(parts) > 1 else "UNKNOWN"


            def count_levels(path):
                counts = {}
                for line in read_all(path):
                    lv = level_of(line)
                    counts[lv] = counts.get(lv, 0) + 1
                return counts
        """,
        "tests/test_levels.py": TEST_LEVELS,
    }
    fix = {
        "logsum/reader.py": """
            def iter_lines(path):
                with open(path, encoding="utf-8", errors="replace") as f:
                    for line in f:
                        yield line.rstrip("\\n")
        """,
        "logsum/levels.py": """
            from .reader import iter_lines


            def level_of(line):
                parts = line.split(" ", 2)
                return parts[1] if len(parts) > 1 else "UNKNOWN"


            def count_levels(path):
                counts = {}
                for line in iter_lines(path):
                    lv = level_of(line)
                    counts[lv] = counts.get(lv, 0) + 1
                return counts
        """,
    }
    n = {".relai/notes.md": notes(
        ["2026-11-06 — log lines look like `2026-11-01T10:00:00 ERROR disk full` (timestamp, level, message)."],
        ["2026-11-06 — production logs are several GB: never read a whole log in memory "
         "(no read_text/read/readlines); go through logsum.reader.iter_lines. The first version did and the job was OOM-killed."],
        ["2026-11-10 — add `top_messages(path, n=3)` to logsum/levels.py: the n most frequent ERROR messages "
         "as a list of (message, count), most frequent first, ties by message. Add a test."])}
    single("s1", d, scaffold, fix, n,
           "fix: stream the log line by line (prod logs are several GB; read_text got the job OOM-killed)")


TEST_LEVELS = """
    import os
    import tempfile
    import unittest

    from logsum.levels import count_levels, level_of


    class T(unittest.TestCase):
        def test_level_of(self):
            self.assertEqual(level_of("t ERROR x y"), "ERROR")
            self.assertEqual(level_of("t"), "UNKNOWN")

        def test_count(self):
            with tempfile.TemporaryDirectory() as t:
                p = os.path.join(t, "a.log")
                with open(p, "w") as f:
                    f.write("t INFO a\\nt ERROR b\\nt INFO c\\n")
                self.assertEqual(count_levels(p), {"INFO": 2, "ERROR": 1})


    if __name__ == "__main__":
        unittest.main()
"""


def s2(d):
    scaffold = {
        "todo/__init__.py": "",
        "todo/store.py": """
            import json
            import os


            def load(path):
                if not os.path.exists(path):
                    return []
                with open(path) as f:
                    return json.load(f)


            def save(path, items):
                with open(path, "w") as f:
                    json.dump(items, f, indent=1)
        """,
        "todo/core.py": """
            from . import store


            def add(path, text):
                items = store.load(path)
                nid = max([i["id"] for i in items], default=0) + 1
                items.append({"id": nid, "text": text, "done": False})
                store.save(path, items)
                return nid


            def list_items(path):
                return store.load(path)
        """,
        "tests/test_core.py": """
            import os
            import tempfile
            import unittest

            from todo import core


            class T(unittest.TestCase):
                def test_add_list_done(self):
                    with tempfile.TemporaryDirectory() as t:
                        p = os.path.join(t, "todo.json")
                        a = core.add(p, "a")
                        b = core.add(p, "b")
                        self.assertEqual((a, b), (1, 2))
                        core.done(p, 1)
                        items = core.list_items(p)
                        self.assertTrue(items[0]["done"])
                        self.assertFalse(items[1]["done"])


            if __name__ == "__main__":
                unittest.main()
        """,
    }
    fix = {"todo/core.py": scaffold["todo/core.py"] + """

            def done(path, item_id):
                items = store.load(path)
                for i in items:
                    if i["id"] == item_id:
                        i["done"] = True
                store.save(path, items)
        """}
    # keep indentation consistent for dedent
    fix["todo/core.py"] = textwrap.dedent(scaffold["todo/core.py"]).lstrip("\n") + textwrap.dedent("""

        def done(path, item_id):
            items = store.load(path)
            for i in items:
                if i["id"] == item_id:
                    i["done"] = True
            store.save(path, items)
    """)
    n = {".relai/notes.md": notes(
        ["2026-11-08 — todo.json is merged between devices by item id (a sync job outside this repo). "
         "Items are therefore never removed from the file: a removed item comes back at the next sync. "
         "Deleting = setting `deleted_at` (ISO timestamp string) on the item; deleted items are hidden from list_items."],
        ["2026-11-08 — ids are per file and must stay unique, so never renumber items."],
        ["2026-11-10 — add `delete(path, item_id)` to todo/core.py. Add a test."])}
    single("s2", d, scaffold, fix, n, "feat: done() marks an item as done")


def s3(d):
    state_v1 = """
        import json


        def save_state(path, state):
            with open(path, "w") as f:
                json.dump(state, f, sort_keys=True)


        def load_state(path):
            with open(path) as f:
                return json.load(f)
    """
    state_v2 = """
        import json
        import os
        import tempfile


        def save_state(path, state):
            d = os.path.dirname(os.path.abspath(path))
            fd, tmp = tempfile.mkstemp(dir=d)
            with os.fdopen(fd, "w") as f:
                json.dump(state, f, sort_keys=True)
            os.replace(tmp, path)


        def load_state(path):
            with open(path) as f:
                return json.load(f)
    """
    scaffold = {
        "dash/__init__.py": "",
        "dash/store.py": state_v1,
        "tests/test_store.py": """
            import os
            import tempfile
            import unittest

            from dash.store import load_state, save_state


            class T(unittest.TestCase):
                def test_roundtrip(self):
                    with tempfile.TemporaryDirectory() as t:
                        p = os.path.join(t, "state.json")
                        save_state(p, {"a": 1})
                        self.assertEqual(load_state(p), {"a": 1})


            if __name__ == "__main__":
                unittest.main()
        """,
    }
    n = {".relai/notes.md": notes(
        ["2026-11-06 — the dashboard (separate process) polls every file under out/ once a second."],
        ["2026-11-06 — a file under out/ must show up complete or not at all: write a temporary file in the same "
         "directory, then os.replace it onto the final path. Never open the final path for writing."],
        ["2026-11-10 — add `export_csv(state, path)` in dash/export.py (new file): header `key,value`, then one row "
         "per key of the state dict, sorted by key. Add a test."])}
    single("s3", d, scaffold, {"dash/store.py": state_v2}, n,
           "fix: write state.json atomically (the dashboard polls out/ every second and crashed on half-written files)")


# ---------------------------------------------------------------- multi repos

def linked(d, name, summary, commits):
    init(d)
    for date, msg, files in commits:
        files = dict(files)
        files.setdefault("relai.yaml", meta(name, summary=summary))
        commit(d, date, msg, files)


def app_repo(d, links, scaffold, nxt, pitfalls=None):
    init(d)
    base = {"AGENTS.md": AGENTS, "relai.yaml": meta("app", links)}
    base.update(scaffold)
    commit(d, "2026-10-25", "feat: first version", base)
    commit(d, "2026-11-01", "docs: session notes", {".relai/notes.md": notes(
        ["2026-10-25 — app is a thin layer over the shared repositories listed in `links`."],
        pitfalls or ["2026-10-25 — tests use tempfile dirs; never write inside the repo."],
        nxt)})


def m1(d):
    os.makedirs(d, exist_ok=True)
    linked(os.path.join(d, "store"), "store", "Record store layout.", [
        ("2026-10-20", "feat: record layout", {
            "store/__init__.py": "",
            "store/layout.py": """
                import os


                def path_for(root, rid):
                    return os.path.join(root, rid + ".json")
            """}),
        ("2026-11-05", "feat!: shard records as <root>/<rid[:2]>/<rid>.json; flat <root>/<rid>.json is no longer written "
                       "(directories with 100k files were too slow)", {
            "store/layout.py": """
                import os


                def path_for(root, rid):
                    return os.path.join(root, rid[:2], rid + ".json")
            """,
            ".relai/notes.md": notes(
                ["2026-11-05 — records are sharded by the first two characters of the id; every reader must use the "
                 "sharded path, the flat files are gone after migration."], ["- none"][:0], []),
        }),
    ])
    linked(os.path.join(d, "tools"), "tools", "Formatting helpers.", [
        ("2026-10-22", "feat: table printer", {"tools/__init__.py": "", "tools/table.py": "def row(*c):\n    return ' | '.join(map(str, c))\n"}),
        ("2026-11-06", "chore: reformat", {"tools/table.py": "def row(*c):\n    return \" | \".join(map(str, c))\n"}),
    ])
    scaffold = {
        "app/__init__.py": "",
        "app/records.py": """
            import json
            import os


            def load_record(root, rid):
                with open(os.path.join(root, rid + ".json")) as f:
                    return json.load(f)
        """,
        "tests/test_records.py": """
            import json
            import os
            import tempfile
            import unittest

            from app.records import load_record


            class T(unittest.TestCase):
                def test_load(self):
                    with tempfile.TemporaryDirectory() as t:
                        with open(os.path.join(t, "ab12.json"), "w") as f:
                            json.dump({"title": "x"}, f)
                        self.assertEqual(load_record(t, "ab12"), {"title": "x"})


            if __name__ == "__main__":
                unittest.main()
        """,
    }
    app_repo(os.path.join(d, "app"), ["store", "tools"], scaffold,
             ["2026-11-01 — add `title_of(root, rid)` to app/records.py: the record's `title`, or `(untitled)` when "
              "it has none. Add a test."])


def m2(d):
    os.makedirs(d, exist_ok=True)
    linked(os.path.join(d, "platform"), "platform", "Service launcher.", [
        ("2026-10-18", "feat: launcher", {
            "platform/__init__.py": "",
            "platform/launcher.py": """
                def env_for(settings):
                    return dict(settings)
            """}),
        ("2026-11-04", "feat!: launcher passes only SVC_-prefixed variables to services (SVC_PORT, SVC_WORKERS, ...); "
                       "bare PORT/WORKERS are dropped", {
            "platform/launcher.py": """
                PREFIX = "SVC_"


                def env_for(settings):
                    return {PREFIX + k.upper(): str(v) for k, v in settings.items()}
            """,
            ".relai/notes.md": notes(
                ["2026-11-04 — services get their settings only as SVC_<NAME> environment variables."], [], []),
        }),
    ])
    linked(os.path.join(d, "ui"), "ui", "Web widgets.", [
        ("2026-10-21", "feat: button", {"ui/__init__.py": ""}),
        ("2026-11-07", "docs: button usage", {"ui/USAGE.txt": "Use the button.\n"}),
    ])
    scaffold = {
        "app/__init__.py": "",
        "app/cfg.py": """
            def get_port(env):
                return int(env.get("PORT", "8080"))
        """,
        "tests/test_cfg.py": """
            import unittest

            from app.cfg import get_port


            class T(unittest.TestCase):
                def test_port(self):
                    self.assertEqual(get_port({}), 8080)
                    self.assertEqual(get_port({"PORT": "9000"}), 9000)


            if __name__ == "__main__":
                unittest.main()
        """,
    }
    app_repo(os.path.join(d, "app"), ["platform"], scaffold,
             ["2026-11-01 — add `get_workers(env)` to app/cfg.py: the worker count as an int, default 1. "
              "Add a test."])


def m3(d):
    os.makedirs(d, exist_ok=True)
    linked(os.path.join(d, "shared-dates"), "shared-dates", "Calendar conventions.", [
        ("2026-10-19", "feat: week helpers", {
            "shared_dates/__init__.py": "",
            "shared_dates/weeks.py": """
                from datetime import timedelta


                def week_start(d):
                    return d - timedelta(days=(d.weekday() + 1) % 7)
            """}),
        ("2026-11-05", "feat!: weeks start on Monday (ISO 8601) in every service; Sunday-start weeks are removed", {
            "shared_dates/weeks.py": """
                from datetime import timedelta


                def week_start(d):
                    return d - timedelta(days=d.weekday())
            """,
            ".relai/notes.md": notes(
                ["2026-11-05 — a week starts on Monday everywhere, reports included. Old Sunday-start weekly "
                 "figures were regenerated."], [], []),
        }),
    ])
    linked(os.path.join(d, "ui"), "ui", "Web widgets.", [
        ("2026-10-21", "feat: button", {"ui/__init__.py": ""}),
        ("2026-11-07", "docs: button usage", {"ui/USAGE.txt": "Use the button.\n"}),
    ])
    scaffold = {
        "app/__init__.py": "",
        "app/report.py": """
            from datetime import timedelta


            def week_of(d):
                return d - timedelta(days=(d.weekday() + 1) % 7)
        """,
        "tests/test_report.py": """
            import unittest
            from datetime import date

            from app.report import week_of


            class T(unittest.TestCase):
                def test_week_of(self):
                    self.assertEqual(week_of(date(2026, 11, 4)), date(2026, 11, 1))


            if __name__ == "__main__":
                unittest.main()
        """,
    }
    app_repo(os.path.join(d, "app"), ["shared-dates", "ui"], scaffold,
             ["2026-11-01 — add `weekly_totals(rows)` to app/report.py: rows are (iso_date_string, int amount); "
              "return {week_start_iso_string: sum of amounts}. Add a test."])


TASKS = {"s1": s1, "s2": s2, "s3": s3, "m1": m1, "m2": m2, "m3": m3}

if __name__ == "__main__":
    if len(sys.argv) != 3 or sys.argv[1] not in TASKS:
        sys.exit("usage: build.py <s1|s2|s3|m1|m2|m3> <dir>")
    TASKS[sys.argv[1]](os.path.abspath(sys.argv[2]))
