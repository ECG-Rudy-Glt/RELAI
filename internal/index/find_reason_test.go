package index

import "testing"

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
