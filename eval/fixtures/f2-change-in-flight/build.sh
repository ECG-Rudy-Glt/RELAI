#!/usr/bin/env bash
# F2 — change in flight. An invoice HTTP API. An OpenSpec change (add-rate-limiting) is 3/5 done
# and its task inv-7 is in progress; a higher-looking P1 chore (inv-9) is ready and unrelated.
# Correct: carry on inv-7, i.e. task 2.1 of add-rate-limiting (429 + Retry-After).
# Trap: inv-9 (logging upgrade), inv-10, or starting rate limiting from scratch.
source "$(dirname "$0")/../lib.sh"
fx_init "$1"

write README.md <<'EOF'
# invoice-api

HTTP API that renders invoices as PDF. `go run ./cmd/server`, then `POST /invoices`.

Specs live in `openspec/`, tasks in Beads (`bd ready`).
EOF
write go.mod <<'EOF'
module example.com/invoice-api

go 1.23
EOF
write cmd/server/main.go <<'EOF'
package main

import (
	"log"
	"net/http"

	"example.com/invoice-api/internal/api"
)

func main() { log.Fatal(http.ListenAndServe(":8080", api.Routes())) }
EOF
write internal/api/routes.go <<'EOF'
// Package api exposes the HTTP routes.
package api

import "net/http"

// Routes returns the HTTP handler of the API.
func Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /invoices", createInvoice)
	return mux
}

func createInvoice(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusCreated) }
EOF
commit "feat: POST /invoices"

write openspec/config.yaml <<'EOF'
schema: spec-driven
EOF
write openspec/specs/invoices/spec.md <<'EOF'
# invoices

## Requirements
### Requirement: Create invoice
The API SHALL create an invoice from a JSON body on `POST /invoices` and answer 201.

#### Scenario: valid body
- **WHEN** a client posts a valid invoice
- **THEN** the API answers 201 with the invoice ID
EOF
write openspec/changes/add-rate-limiting/proposal.md <<'EOF'
## Why
A customer script sent 40,000 invoices in ten minutes and took the PDF workers down (2026-06-03).

## What Changes
- Token-bucket rate limit per API key on `POST /invoices` (60/min, burst 10).
- Over the limit: `429 Too Many Requests` with a `Retry-After` header.
- Limits configurable per key.

## Impact
- `internal/api`, new `internal/ratelimit`.
EOF
write openspec/changes/add-rate-limiting/specs/invoices/spec.md <<'EOF'
## ADDED Requirements
### Requirement: Rate limit per API key
The API SHALL limit `POST /invoices` to 60 requests per minute per API key, with a burst of 10.

#### Scenario: over the limit
- **WHEN** a key exceeds its limit
- **THEN** the API answers 429 with a `Retry-After` header in seconds
EOF
write openspec/changes/add-rate-limiting/tasks.md <<'EOF'
## 1. Limiter
- [ ] 1.1 Token bucket in `internal/ratelimit`
- [ ] 1.2 Per-key buckets with idle eviction
- [ ] 1.3 Unit tests (refill, burst, eviction)

## 2. HTTP
- [ ] 2.1 Middleware on `POST /invoices`: 429 + `Retry-After`
- [ ] 2.2 Per-key limits from config
EOF
bd_init inv
task inv-7 1 "Rate limiting on POST /invoices (openspec: add-rate-limiting)"
task inv-9 1 "Upgrade logging library to v2"
task inv-10 2 "Add OpenAPI description for /invoices"
task inv-11 3 "Dark mode for the PDF preview page"
start_task inv-7
commit "docs: openspec change add-rate-limiting, task graph"

write internal/ratelimit/bucket.go <<'EOF'
// Package ratelimit implements per-key token buckets.
package ratelimit

import (
	"sync"
	"time"
)

// Bucket is a token bucket: Rate tokens per second, at most Burst tokens.
type Bucket struct {
	Rate, Burst float64
	tokens      float64
	last        time.Time
}

// Allow takes one token if available.
func (b *Bucket) Allow(now time.Time) bool {
	if b.last.IsZero() {
		b.tokens, b.last = b.Burst, now
	}
	b.tokens = min(b.Burst, b.tokens+now.Sub(b.last).Seconds()*b.Rate)
	b.last = now
	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Limiter holds one bucket per key and evicts idle ones.
type Limiter struct {
	mu      sync.Mutex
	buckets map[string]*Bucket
}
EOF
sed -i 's/- \[ \] 1.1/- [x] 1.1/' openspec/changes/add-rate-limiting/tasks.md
commit "feat(ratelimit): token bucket (add-rate-limiting 1.1)"

sed -i 's/- \[ \] 1.2/- [x] 1.2/' openspec/changes/add-rate-limiting/tasks.md
commit "feat(ratelimit): per-key buckets with idle eviction (1.2)"

write internal/ratelimit/bucket_test.go <<'EOF'
package ratelimit

import (
	"testing"
	"time"
)

func TestBurstThenRefill(t *testing.T) {
	b := &Bucket{Rate: 1, Burst: 2}
	now := time.Unix(0, 0)
	if !b.Allow(now) || !b.Allow(now) || b.Allow(now) {
		t.Fatal("burst of 2 expected")
	}
	if !b.Allow(now.Add(time.Second)) {
		t.Fatal("one token after one second")
	}
}
EOF
sed -i 's/- \[ \] 1.3/- [x] 1.3/' openspec/changes/add-rate-limiting/tasks.md
commit "test(ratelimit): refill, burst, eviction (1.3)"

write docs/journal.md <<'EOF'
# Journal

## 2026-06-03
Incident: a customer script sent 40k invoices, PDF workers down for 25 minutes. Opened the
OpenSpec change add-rate-limiting; inv-7 tracks it.

## 2026-06-06
Limiter package done and tested (tasks 1.1–1.3). Next session: the HTTP middleware (2.1).
Pitfall: the API key is in the `X-Api-Key` header, not `Authorization` — the middleware must key
on that. The logging upgrade (inv-9) waits until rate limiting ships, it touches every handler.
EOF
commit "docs: journal"
