#!/usr/bin/env bash
# F1 — stale docs. A Go link checker. TODO.md still says the next step is retries (lc-3), but
# retries were merged afterwards; the task graph says lc-5 (worker pool) is next, and the
# journal explains why it comes before the HTML report.
# Correct: lc-5. Trap: lc-3 (done) or lc-4 (blocked by lc-5).
source "$(dirname "$0")/../lib.sh"
fx_init "$1"

write README.md <<'EOF'
# linkcheck

Checks every link of a website and reports the broken ones.

    go run ./cmd/linkcheck https://example.com

See `TODO.md` for the plan and `docs/journal.md` for the history.
EOF
write go.mod <<'EOF'
module example.com/linkcheck

go 1.23
EOF
write cmd/linkcheck/main.go <<'EOF'
package main

import (
	"fmt"
	"os"

	"example.com/linkcheck/internal/crawl"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: linkcheck URL")
		os.Exit(2)
	}
	for _, r := range crawl.Run(os.Args[1]) {
		fmt.Println(r)
	}
}
EOF
write internal/crawl/crawl.go <<'EOF'
// Package crawl fetches pages one at a time and reports broken links.
package crawl

import "net/http"

// Run checks the links reachable from start. Sequential for now.
func Run(start string) []string {
	resp, err := http.Get(start)
	if err != nil {
		return []string{"BROKEN " + start + ": " + err.Error()}
	}
	resp.Body.Close()
	return nil
}
EOF
commit "feat: sequential crawler"

write TODO.md <<'EOF'
# TODO

Next step: **add retries with exponential backoff (lc-3)** — flaky hosts make the report noisy.

- [x] Sequential crawler
- [ ] Retries with backoff (lc-3)
- [ ] HTML report (lc-4)
- [ ] Respect robots.txt (lc-7)
EOF
write docs/journal.md <<'EOF'
# Journal

## 2026-06-02
Sequential crawler works on small sites. Planned: retries (lc-3), then HTML report (lc-4).
EOF
bd_init lc
task lc-3 1 "Retries with exponential backoff for flaky hosts"
task lc-5 1 "Parallel fetching with a bounded worker pool"
task lc-4 1 "HTML report with per-link timings" --deps blocked-by:lc-5
task lc-6 3 "Fix typo in README usage line"
task lc-7 2 "Respect robots.txt"
commit "docs: TODO, journal, task graph"

write internal/crawl/retry.go <<'EOF'
package crawl

import "time"

// backoff returns the wait before attempt n (0-based): 200ms, 400ms, 800ms…
func backoff(n int) time.Duration { return 200 * time.Millisecond << n }
EOF
done_task lc-3 "merged: retry.go"
commit "feat: retries with exponential backoff (closes lc-3)"

write docs/journal.md <<'EOF'
# Journal

## 2026-06-02
Sequential crawler works on small sites. Planned: retries (lc-3), then HTML report (lc-4).

## 2026-06-04
Retries merged (lc-3). Tried the HTML report: per-link timings are meaningless while fetching is
sequential, a 2,000-link site takes 9 minutes. Decision: do the worker pool (lc-5) first, the
report (lc-4) depends on it. robots.txt (lc-7) can wait, nobody asked for it yet.
EOF
commit "docs: journal, worker pool before the report"
