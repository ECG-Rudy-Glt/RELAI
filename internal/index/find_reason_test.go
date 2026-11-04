package index

import (
	"strings"
	"testing"
)

func TestFind_IsCaseInsensitive(t *testing.T) {
	entries := []Entry{{Source: "a", RelaiYAML: Manifest{Provides: []Capability{{Capability: "Auth.OIDC"}}}}}
	if got := Find(entries, "auth.oidc"); len(got) != 1 {
		t.Errorf("Find should ignore case, got %v", got)
	}
}

func TestGitReason(t *testing.T) {
	if got := gitReason("Cloning...\nfatal: repository 'x' not found\x1b[0m\n"); got != ": fatal: repository 'x' not found[0m" {
		t.Errorf("gitReason = %q", got)
	}
	if gitReason("") != "" {
		t.Error("empty stderr should give no reason")
	}
}

func TestLinksAreParsedInAnyListShape(t *testing.T) {
	for name, src := range map[string]string{
		"block":  "id: x\nlinks:\n  - portfolio   # strong: 3 files\n  - shelfmark\n",
		"inline": "id: x\nlinks: [portfolio, shelfmark]\n",
	} {
		m, warn := parseManifest([]byte(src))
		if len(m.Links) != 2 || m.Links[0] != "portfolio" || m.Links[1] != "shelfmark" || len(warn) != 0 {
			t.Errorf("%s: links = %v warn = %v", name, m.Links, warn)
		}
	}
}

func TestDuplicateKeyKeepsBothListsAndWarns(t *testing.T) {
	m, warn := parseManifest([]byte("id: x\nprovides:\n  - capability: d.one\nsummary: s\nprovides:\n  - capability: d.two\n"))
	if len(m.Provides) != 2 || m.Provides[0].Capability != "d.one" || m.Provides[1].Capability != "d.two" {
		t.Errorf("provides = %v: the first list must not be lost", m.Provides)
	}
	if len(warn) == 0 || !strings.Contains(warn[0], "twice") {
		t.Errorf("warnings = %v, want one about the duplicated key", warn)
	}
}

func TestFindOrdersByDeclaredLinksAndNamesTheRelation(t *testing.T) {
	a := Entry{Source: "a", RelaiYAML: Manifest{ID: "alpha"}}
	b := Entry{Source: "b", RelaiYAML: Manifest{ID: "beta", Links: []string{"me"}}}
	c := Entry{Source: "c", RelaiYAML: Manifest{ID: "gamma"}}
	d := Entry{Source: "d", RelaiYAML: Manifest{ID: "delta", Links: []string{"me"}}}
	me := Manifest{ID: "me", Links: []string{"gamma", "delta"}}
	got := Order([]Entry{a, b, c, d}, me)
	var ids []string
	for _, e := range got {
		ids = append(ids, e.RelaiYAML.ID)
	}
	if strings.Join(ids, ",") != "beta,gamma,delta,alpha" {
		t.Errorf("order = %v, want the related ones first in their original order, then the rest", ids)
	}
	for id, want := range map[string]string{"alpha": "", "beta": "links to this repo", "gamma": "linked from here", "delta": "linked both ways"} {
		for _, e := range got {
			if e.RelaiYAML.ID == id && Relation(e, me) != want {
				t.Errorf("%s: relation %q, want %q", id, Relation(e, me), want)
			}
		}
	}
	if len(Order(nil, me)) != 0 || Relation(a, Manifest{}) != "" {
		t.Error("an unknown repo (no id) must relate to nothing")
	}
}
