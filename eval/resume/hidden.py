#!/usr/bin/env python3
"""Hidden acceptance tests. usage: hidden.py <task> <dir>. Prints PASS or FAIL: <why>."""
import importlib, os, re, sys
task, root = sys.argv[1], sys.argv[2]
os.chdir(root); sys.path.insert(0, root)
def fail(m): print("FAIL:", m); sys.exit(0)
try:
    if task == "rename":
        import calc
        if calc.order_total([{'price': 2, 'qty': 3}]) != 6: fail("order_total")
        if not hasattr(calc, "price_total") or calc.price_total([{'price': 2, 'qty': 3}]) != 6: fail("deprecated alias price_total is gone")
        for n in "abc":
            src = open(f"{n}.py").read()
            if "price_total" in src: fail(f"{n}.py still uses price_total")
            if importlib.import_module(n).run([{'price': 1, 'qty': 1}]) != 1: fail(f"{n}.run")
    elif task == "stale":
        import export
        if export.export_csv([['a,b', 'c"d']]) != '"a,b","c""d"\n': fail("csv export changed")
        if not hasattr(export, "export_json"): fail("no export_json")
        import json
        if json.loads(export.export_json([[1, "x"]])) not in ([[1, "x"]], [{"0": 1, "1": "x"}]): fail("export_json output")
    elif task == "dirty":
        import text
        for s, want in [("Café Noël", "café-noël"), ("  Hello,  World! ", "hello-world"), ("--a--", "a")]:
            if text.slugify(s) != want: fail(f"slugify({s!r}) = {text.slugify(s)!r}")
    elif task == "money":
        import ledger
        from decimal import Decimal
        r = ledger.refund('acct', '0.10'); r2 = ledger.refund('acct', '0.20')
        if not isinstance(r2, Decimal) or r2 != Decimal('100.30'): fail(f"refund result {r2!r}")
except SystemExit: raise
except Exception as e: fail(f"{type(e).__name__}: {e}")
print("PASS")
