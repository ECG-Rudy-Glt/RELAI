### Decisions
- 2026-06-02 — Postgres 17 before the currency backfill — the backfill uses JSON_TABLE.

### Pitfalls
- `invoices` has 40M rows: data migrations must be batched.
