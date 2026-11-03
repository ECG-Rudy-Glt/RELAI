#!/usr/bin/env bash
# F4 — stale tracker. A billing service. The Postgres 17 migration (pg-4) was merged and verified,
# but nobody closed the task: `bd ready` still lists it first, and the backfill (pg-6) shows as
# blocked by it. Only the commits and the journal tell the truth.
# Correct: pg-6 (close pg-4, then backfill). Trap: redo pg-4, or pick pg-5 / pg-7.
source "$(dirname "$0")/../lib.sh"
fx_init "$1"

write README.md <<'EOF'
# billing

Billing service: subscriptions, invoices, payments. Go + Postgres.
Tasks are tracked in Beads (`bd ready`); history in `docs/journal.md`.
EOF
write go.mod <<'EOF'
module example.com/billing

go 1.23
EOF
write internal/db/db.go <<'EOF'
// Package db opens the Postgres connection.
package db

// MinServerVersion is the oldest Postgres the service runs on.
const MinServerVersion = 14
EOF
write migrations/0001_init.sql <<'EOF'
CREATE TABLE invoices (id bigserial PRIMARY KEY, customer_id bigint NOT NULL, total_cents bigint NOT NULL);
EOF
commit "feat: invoices table"

bd_init pg
task pg-4 0 "Migrate the database from Postgres 14 to 17"
task pg-6 1 "Backfill invoices.currency for rows created before multi-currency" --deps blocked-by:pg-4
task pg-5 2 "Add a /healthz endpoint"
task pg-7 3 "Rename internal package util to money"
write docs/journal.md <<'EOF'
# Journal

## 2026-06-02
Multi-currency needs Postgres 17 (we rely on its JSON_TABLE for the backfill). Plan: migrate (pg-4),
then backfill invoices.currency (pg-6). The health endpoint (pg-5) is nice to have.
EOF
commit "docs: plan Postgres 17 then backfill"

write migrations/0002_currency.sql <<'EOF'
ALTER TABLE invoices ADD COLUMN currency char(3);
EOF
commit "feat: invoices.currency column (nullable until the backfill)"

sed -i 's/MinServerVersion = 14/MinServerVersion = 17/' internal/db/db.go
write ops/pg17-upgrade.md <<'EOF'
# Postgres 14 → 17 upgrade (pg-4)

Done 2026-06-05 with pg_upgrade --link on staging, then production at 06:00.
Checks: row counts equal, app smoke tests green, replication caught up. Old cluster kept 7 days.
EOF
commit "feat: run on Postgres 17, upgrade runbook (pg-4)"

cat >>docs/journal.md <<'EOF'

## 2026-06-05
Production is on Postgres 17 since 06:00, all checks green (ops/pg17-upgrade.md). Forgot to close
pg-4 in Beads before leaving. Next: the currency backfill (pg-6); write it as a batched migration,
10k rows per batch, invoices is 40M rows.
EOF
commit "docs: journal, Postgres 17 in production"
