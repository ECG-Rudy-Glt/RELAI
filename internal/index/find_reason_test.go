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
