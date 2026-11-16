package index

import "testing"

func capList(names ...string) []Capability {
	var c []Capability
	for _, n := range names {
		c = append(c, Capability{Capability: n})
	}
	return c
}

func TestFindConsumersMatchAndNoMatch(t *testing.T) {
	entries := []Entry{
		{RelaiYAML: Manifest{ID: "web", Consumes: capList("auth.oidc", "db.pg")}},
		{RelaiYAML: Manifest{ID: "auth", Provides: capList("auth.oidc")}},
	}
	got := FindConsumers(entries, "auth.oidc")
	if len(got) != 1 || got[0].RelaiYAML.ID != "web" {
		t.Fatalf("want only web, got %+v", got)
	}
	if got := FindConsumers(entries, "nope"); len(got) != 0 {
		t.Errorf("want none, got %+v", got)
	}
}

func TestFindConsumersRepoThatProvidesAndConsumes(t *testing.T) {
	entries := []Entry{{RelaiYAML: Manifest{ID: "gateway", Provides: capList("api.edge"), Consumes: capList("auth.oidc")}}}
	if len(FindConsumers(entries, "auth.oidc")) != 1 || len(Find(entries, "auth.oidc")) != 0 {
		t.Error("consumes: must only be found by FindConsumers")
	}
	if len(Find(entries, "api.edge")) != 1 || len(FindConsumers(entries, "api.edge")) != 0 {
		t.Error("provides: must only be found by Find")
	}
}

func TestFindConsumersMatchingRule(t *testing.T) {
	entries := []Entry{{RelaiYAML: Manifest{ID: "web", Consumes: capList("Auth.OIDC", "auth.oidc")}}}
	if got := FindConsumers(entries, "auth.OidC"); len(got) != 1 {
		t.Errorf("case-insensitive, one entry even when listed twice: %+v", got)
	}
	// exact name only: no prefix, no trimming
	for _, q := range []string{"auth", "auth.oidc.v2", " auth.oidc"} {
		if got := FindConsumers(entries, q); len(got) != 0 {
			t.Errorf("%q must not match: %+v", q, got)
		}
	}
	if got := FindConsumers(nil, "x"); got != nil {
		t.Errorf("empty index: %+v", got)
	}
}
