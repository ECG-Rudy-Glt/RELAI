#!/usr/bin/env python3
"""Usage: solutions.py <task> <dir>. Applies a reference solution (fixture check only)."""
import os
import sys


def append(path, text):
    with open(path, "a") as f:
        f.write(text)


def put(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(text)


def s1(d):
    append(d + "/logsum/levels.py", '''

def top_messages(path, n=3):
    counts = {}
    for line in iter_lines(path):
        parts = line.split(" ", 2)
        if len(parts) == 3 and parts[1] == "ERROR":
            counts[parts[2]] = counts.get(parts[2], 0) + 1
    return sorted(counts.items(), key=lambda kv: (-kv[1], kv[0]))[:n]
''')
    put(d + "/tests/test_top.py", '''import os
import tempfile
import unittest

from logsum.levels import top_messages


class T(unittest.TestCase):
    def test_top(self):
        with tempfile.TemporaryDirectory() as t:
            p = os.path.join(t, "a.log")
            with open(p, "w") as f:
                f.write("t ERROR a\\nt ERROR b\\nt ERROR a\\nt INFO a\\n")
            self.assertEqual(top_messages(p, 1), [("a", 2)])
''')


def s2(d):
    p = d + "/todo/core.py"
    s = open(p).read().replace("    return store.load(path)\n",
                               '    return [i for i in store.load(path) if not i.get("deleted_at")]\n', 1)
    s = "import datetime\n" + s + '''

def delete(path, item_id):
    items = store.load(path)
    for i in items:
        if i["id"] == item_id and not i.get("deleted_at"):
            i["deleted_at"] = datetime.datetime.now(datetime.timezone.utc).isoformat()
    store.save(path, items)
'''
    put(p, s)
    put(d + "/tests/test_delete.py", '''import os
import tempfile
import unittest

from todo import core


class T(unittest.TestCase):
    def test_delete(self):
        with tempfile.TemporaryDirectory() as t:
            p = os.path.join(t, "todo.json")
            core.add(p, "a")
            core.add(p, "b")
            core.delete(p, 1)
            self.assertEqual([i["text"] for i in core.list_items(p)], ["b"])
''')


def s3(d):
    put(d + "/dash/export.py", '''import csv
import os
import tempfile


def export_csv(state, path):
    dirname = os.path.dirname(os.path.abspath(path))
    fd, tmp = tempfile.mkstemp(dir=dirname)
    with os.fdopen(fd, "w", newline="") as f:
        w = csv.writer(f)
        w.writerow(["key", "value"])
        for k in sorted(state):
            w.writerow([k, state[k]])
    os.replace(tmp, path)
''')
    put(d + "/tests/test_export.py", '''import os
import tempfile
import unittest

from dash.export import export_csv


class T(unittest.TestCase):
    def test_export(self):
        with tempfile.TemporaryDirectory() as t:
            p = os.path.join(t, "o.csv")
            export_csv({"b": 1, "a": 2}, p)
            self.assertIn("a,2", open(p).read())
''')


def m1(d):
    d += "/app"
    put(d + "/app/records.py", '''import json
import os


def load_record(root, rid):
    with open(os.path.join(root, rid[:2], rid + ".json")) as f:
        return json.load(f)


def title_of(root, rid):
    return load_record(root, rid).get("title", "(untitled)")
''')
    t = open(d + "/tests/test_records.py").read().replace(
        'with open(os.path.join(t, "ab12.json"), "w")',
        'os.makedirs(os.path.join(t, "ab"))\n            with open(os.path.join(t, "ab", "ab12.json"), "w")')
    put(d + "/tests/test_records.py", t)


def m2(d):
    d += "/app"
    put(d + "/app/cfg.py", '''def get_port(env):
    return int(env.get("SVC_PORT", "8080"))


def get_workers(env):
    return int(env.get("SVC_WORKERS", "1"))
''')
    t = open(d + "/tests/test_cfg.py").read().replace('"PORT"', '"SVC_PORT"')
    put(d + "/tests/test_cfg.py", t)


def m3(d):
    d += "/app"
    put(d + "/app/report.py", '''from datetime import date, timedelta


def week_of(d):
    return d - timedelta(days=d.weekday())


def weekly_totals(rows):
    out = {}
    for iso, amount in rows:
        k = week_of(date.fromisoformat(iso)).isoformat()
        out[k] = out.get(k, 0) + amount
    return out
''')
    t = open(d + "/tests/test_report.py").read().replace("date(2026, 11, 1))", "date(2026, 11, 2))")
    put(d + "/tests/test_report.py", t)


TASKS = {"s1": s1, "s2": s2, "s3": s3, "m1": m1, "m2": m2, "m3": m3}

if __name__ == "__main__":
    TASKS[sys.argv[1]](os.path.abspath(sys.argv[2]))
