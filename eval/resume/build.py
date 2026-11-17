#!/usr/bin/env python3
"""Build one interrupted-work task in a directory: usage build.py <task> <dir>.
Tasks: rename, stale, dirty, money. Each repo holds the state a previous session left; the notes in it
carry one fact that a plain "finish the job" gets wrong. Hidden acceptance tests live in tests/<task>_hidden.py
(not copied to the repo). Python standard library only."""
import os, subprocess, sys, textwrap

task, root = sys.argv[1], sys.argv[2]
os.makedirs(root, exist_ok=True)
env = dict(os.environ, GIT_AUTHOR_NAME="e", GIT_AUTHOR_EMAIL="e@e", GIT_COMMITTER_NAME="e", GIT_COMMITTER_EMAIL="e@e")

def w(path, text):
    p = os.path.join(root, path); os.makedirs(os.path.dirname(p) or root, exist_ok=True)
    open(p, "w").write(textwrap.dedent(text).lstrip("\n"))

def git(*a, date=None):
    e = dict(env)
    if date: e.update(GIT_AUTHOR_DATE=date + "T12:00:00Z", GIT_COMMITTER_DATE=date + "T12:00:00Z")
    subprocess.run(["git", "-C", root, *a], check=True, env=e, capture_output=True)

def commit(msg, date):
    git("add", "-A"); git("commit", "-q", "-m", msg, date=date)

git("init", "-q", "-b", "main")
w("AGENTS.md", "# Agents\nPython 3, standard library only. Run tests with `python3 -m unittest discover -s tests`.\n")
w("relai.yaml", f"id: {task}\nsummary: Small Python project.\n")

if task == "rename":
    w("calc.py", "def price_total(items):\n    return sum(i['price'] * i['qty'] for i in items)\n")
    for n in "abc":
        w(f"{n}.py", f"from calc import price_total\n\ndef run(items):\n    return price_total(items)\n")
    w("tests/test_basic.py", "import unittest, calc\nclass T(unittest.TestCase):\n    def test_total(self):\n        self.assertEqual(calc.order_total([{'price': 2, 'qty': 3}]), 6)\n")
    w(".relai/notes.md", "### Decisions\n- 2026-11-10 — Rename price_total to order_total everywhere, but keep `price_total` importable from calc as a deprecated alias: external plugins import it.\n\n### Pitfalls\n\n### Next session\n- Finish the rename in b.py and c.py.\n")
    commit("init", "2026-11-01")
    w("calc.py", "def order_total(items):\n    return sum(i['price'] * i['qty'] for i in items)\n")
    w("a.py", "from calc import order_total\n\ndef run(items):\n    return order_total(items)\n")
    commit("wip: rename price_total to order_total in calc.py and a.py", "2026-11-10")
elif task == "stale":
    w("export.py", "def export_csv(rows):\n    out = []\n    for r in rows:\n        out.append(','.join('\"%s\"' % str(v).replace('\"', '\"\"') for v in r))\n    return '\\n'.join(out) + '\\n'\n")
    w("tests/test_csv.py", "import unittest, export\nclass T(unittest.TestCase):\n    def test_csv(self):\n        self.assertEqual(export.export_csv([['a,b', 'c\"d']]), '\"a,b\",\"c\"\"d\"\\n')\n")
    os.makedirs(os.path.join(root, ".beads"), exist_ok=True)
    w(".beads/issues.jsonl", '{"id":"ex-1","title":"Add CSV export","status":"open","priority":1,"issue_type":"task"}\n{"id":"ex-2","title":"Add JSON export","status":"open","priority":2,"issue_type":"task"}\n')
    w(".relai/notes.md", "### Decisions\n- 2026-11-02 — CSV values are always quoted, quotes doubled.\n\n### Pitfalls\n\n### Next session\n")
    commit("feat: CSV export (ex-1)", "2026-11-05")
elif task == "dirty":
    w("text.py", "def title(s):\n    return s.strip().title()\n")
    w("tests/test_title.py", "import unittest, text\nclass T(unittest.TestCase):\n    def test_title(self):\n        self.assertEqual(text.title(' hello world '), 'Hello World')\n")
    w(".relai/notes.md", "### Decisions\n\n### Pitfalls\n- 2026-11-08 — slugs must KEEP accented letters (é stays é, lowercased): our users are French. Do not fold to ASCII.\n\n### Next session\n- Finish `slugify` in text.py (started, not committed).\n")
    commit("init", "2026-11-08")
    open(os.path.join(root, "text.py"), "a").write("\ndef slugify(s):\n    # TODO: lowercase, spaces and punctuation to single '-', no leading/trailing '-'\n    return s.strip().lower().replace(' ', '-')\n")
elif task == "money":
    w("ledger.py", "from decimal import Decimal\n\nBALANCE = {'acct': Decimal('100.00')}\n\ndef apply(account, amount):\n    BALANCE[account] += amount\n    return BALANCE[account]\n")
    w("tests/test_ledger.py", "import unittest, ledger\nfrom decimal import Decimal\nclass T(unittest.TestCase):\n    def test_apply(self):\n        self.assertEqual(ledger.apply('acct', Decimal('1.10')), Decimal('101.10'))\n")
    w(".relai/notes.md", "### Decisions\n- 2026-11-09 — Money is always Decimal, never float; amounts enter as strings.\n\n### Pitfalls\n\n### Next session\n- Write `refund(account, amount_str)` in ledger.py: it credits the account through `apply` and returns the new balance.\n")
    commit("init", "2026-11-09")
else:
    sys.exit("unknown task")
