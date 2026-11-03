package main

import "testing"

func TestNextLine(t *testing.T) {
	got := NextLine("blah\nNEXT: draft\nmore\n**NEXT: `lc-5` — worker pool**\n")
	if got != "lc-5 — worker pool" {
		t.Fatalf("got %q", got)
	}
	if NextLine("no answer") != "" {
		t.Fatal("expected empty")
	}
}

func TestGrade(t *testing.T) {
	e := Expected{
		AcceptIDs:      []string{"lc-5"},
		AcceptPatterns: []string{"(?i)worker.?pool"},
		Traps:          map[string]string{"lc-3": "stale", "(?i)\\bblog\\b": "dropped"},
	}
	cases := []struct {
		next    string
		correct bool
		trap    string
	}{
		{"lc-5 — parallel fetching", true, ""},
		{"LC-5", true, ""},
		{"lc-3 — add a worker pool for retries", false, "stale"}, // a trap ID beats a pattern
		{"lc-35 — something", false, "other: lc-35 — something"},
		{"build the worker pool", true, ""},
		{"start the blog", false, "dropped"},
		{"worker pool first, the blog can wait", true, ""},
		{"", false, "no NEXT line"},
	}
	for _, c := range cases {
		ok, trap := e.Grade(c.next)
		if ok != c.correct || trap != c.trap {
			t.Errorf("Grade(%q) = %v, %q; want %v, %q", c.next, ok, trap, c.correct, c.trap)
		}
	}
}
