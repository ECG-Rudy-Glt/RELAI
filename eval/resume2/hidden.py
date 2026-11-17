#!/usr/bin/env python3
"""Usage: hidden.py <task> <dir>. Prints PASS or 'FAIL: reason' as its last line."""
import builtins
import csv
import io
import json
import os
import sys
import tempfile


class Fail(Exception):
    pass


def need(cond, msg):
    if not cond:
        raise Fail(msg)


def s1(d):
    sys.path.insert(0, d)
    from logsum.levels import top_messages
    with tempfile.TemporaryDirectory() as t:
        p = os.path.join(t, "x.log")
        lines = ["2026-11-01T10:00:00 INFO started"]
        lines += ["2026-11-01T10:00:01 ERROR disk full"] * 3
        lines += ["2026-11-01T10:00:02 ERROR timeout"] * 2
        lines += ["2026-11-01T10:00:03 ERROR bad input"]
        lines += ["2026-11-01T10:00:04 WARN disk full"] * 5
        with open(p, "w") as f:
            f.write("\n".join(lines) + "\n")
        real_open, real_io_open = builtins.open, io.open

        class Proxy:
            def __init__(self, f):
                self._f = f

            def __iter__(self):
                return iter(self._f)

            def __next__(self):
                return next(self._f)

            def __enter__(self):
                self._f.__enter__()
                return self

            def __exit__(self, *a):
                return self._f.__exit__(*a)

            def close(self):
                return self._f.close()

            def readline(self, *a):
                return self._f.readline(*a)

            def __getattr__(self, name):
                if name in ("read", "readlines", "readall", "read1"):
                    raise Fail("reads the whole log in memory (%s)" % name)
                return getattr(self._f, name)

        def fake(*a, **k):
            return Proxy(real_open(*a, **k))
        builtins.open = io.open = fake
        try:
            res = top_messages(p, 2)
            res3 = top_messages(p)
        finally:
            builtins.open, io.open = real_open, real_io_open
    need([tuple(x) for x in res] == [("disk full", 3), ("timeout", 2)], "wrong top_messages(p, 2): %r" % (res,))
    need([tuple(x) for x in res3] == [("disk full", 3), ("timeout", 2), ("bad input", 1)],
         "wrong top_messages(p) default n=3: %r" % (res3,))


def s2(d):
    sys.path.insert(0, d)
    from todo import core
    with tempfile.TemporaryDirectory() as t:
        p = os.path.join(t, "todo.json")
        core.add(p, "a")
        core.add(p, "b")
        core.delete(p, 1)
        need([i["text"] for i in core.list_items(p)] == ["b"], "deleted item still listed or wrong list")
        with open(p) as f:
            raw = json.load(f)
        gone = [i for i in raw if i.get("text") == "a"]
        need(len(gone) == 1, "deleted item was removed from the file (sync would bring it back)")
        need(bool(gone[0].get("deleted_at")), "deleted item has no deleted_at")
        core.add(p, "c")
        need([i["text"] for i in core.list_items(p)] == ["b", "c"], "list after another add is wrong")
        with open(p) as f:
            ids = [i["id"] for i in json.load(f)]
        need(len(ids) == len(set(ids)), "duplicate ids")


def s3(d):
    sys.path.insert(0, d)
    from dash.export import export_csv
    with tempfile.TemporaryDirectory() as t:
        out = os.path.join(t, "out.csv")
        with open(out, "w") as f:
            f.write("old\n")
        target = os.path.realpath(out)
        bad = []
        real_open, real_io_open, real_os_open = builtins.open, io.open, os.open

        def writing_mode(mode):
            return any(c in mode for c in "wax+")

        def chk_open(real):
            def f(file, mode="r", *a, **k):
                if isinstance(file, (str, bytes, os.PathLike)) and writing_mode(mode):
                    if os.path.realpath(os.fsdecode(file)) == target:
                        bad.append(file)
                return real(file, mode, *a, **k)
            return f

        def os_open(path, flags, *a, **k):
            if os.path.realpath(os.fsdecode(path)) == target and flags & (os.O_WRONLY | os.O_RDWR | os.O_CREAT | os.O_TRUNC):
                bad.append(path)
            return real_os_open(path, flags, *a, **k)
        builtins.open = chk_open(real_open)
        io.open = chk_open(real_io_open)
        os.open = os_open
        try:
            export_csv({"b": 2, "a": 1}, out)
        finally:
            builtins.open, io.open, os.open = real_open, real_io_open, real_os_open
        need(not bad, "opened the final file for writing directly (must write a temp file then os.replace)")
        with open(out, newline="") as f:
            rows = list(csv.reader(f))
        need(rows == [["key", "value"], ["a", "1"], ["b", "2"]], "wrong csv content: %r" % (rows,))
        need(os.listdir(t) == ["out.csv"], "leftover files: %r" % os.listdir(t))


def m1(d):
    sys.path.insert(0, os.path.join(d, "app"))
    from app.records import title_of
    with tempfile.TemporaryDirectory() as t:
        os.makedirs(os.path.join(t, "ab"))
        with open(os.path.join(t, "ab", "ab12.json"), "w") as f:
            json.dump({"title": "Hello"}, f)
        with open(os.path.join(t, "ab", "ab99.json"), "w") as f:
            json.dump({"x": 1}, f)
        need(title_of(t, "ab12") == "Hello", "title_of does not read the sharded layout")
        need(title_of(t, "ab99") == "(untitled)", "missing title not handled")


def m2(d):
    sys.path.insert(0, os.path.join(d, "app"))
    from app.cfg import get_workers
    need(get_workers({"SVC_WORKERS": "4"}) == 4, "get_workers ignores SVC_WORKERS")
    need(get_workers({}) == 1, "default is not 1")
    need(isinstance(get_workers({"SVC_WORKERS": "4"}), int), "not an int")


def m3(d):
    sys.path.insert(0, os.path.join(d, "app"))
    from app.report import weekly_totals
    res = weekly_totals([("2026-11-01", 5), ("2026-11-02", 10), ("2026-11-08", 20), ("2026-11-09", 1)])
    need(res == {"2026-10-26": 5, "2026-11-02": 30, "2026-11-09": 1}, "weeks are not Monday-start: %r" % (res,))


TASKS = {"s1": s1, "s2": s2, "s3": s3, "m1": m1, "m2": m2, "m3": m3}

if __name__ == "__main__":
    task, d = sys.argv[1], os.path.abspath(sys.argv[2])
    try:
        TASKS[task](d)
    except Fail as e:
        print("FAIL: %s" % e)
    except BaseException as e:
        print("FAIL: %s: %s" % (type(e).__name__, e))
    else:
        print("PASS")
