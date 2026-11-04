package index

import (
	"reflect"
	"strings"
	"testing"
)

func caps(m Manifest) []string {
	var out []string
	for _, c := range m.Provides {
		out = append(out, c.Capability)
	}
	return out
}

func TestParseManifest_EveryListShapeGivesTheSameCapabilities(t *testing.T) {
	want := []string{"auth.oidc", "auth.session"}
	cases := map[string]string{
		"block":            "provides:\n  - capability: auth.oidc\n  - capability: auth.session\n",
		"block at col 0":   "provides:\n- capability: auth.oidc\n- capability: auth.session\n",
		"flow maps":        "provides: [{capability: auth.oidc}, {capability: auth.session}]\n",
		"flow scalars":     "provides: [auth.oidc, auth.session]\n",
		"flow quoted":      "provides: [\"auth.oidc\", 'auth.session']  # comment\n",
		"scalar items":     "provides:\n  - auth.oidc\n  - auth.session\n",
		"comment inside":   "provides:\n  # the first one\n  - capability: auth.oidc\n\n  - capability: auth.session\n",
		"capability later": "provides:\n  - description: login\n    capability: auth.oidc\n  - note: x\n    capability: auth.session\n",
		"two keys inline":  "provides:\n  - capability: auth.oidc\n    description: login\n  - capability: auth.session\n",
		"crlf":             "provides:\r\n  - capability: auth.oidc\r\n  - capability: auth.session\r\n",
		"bom":              "\ufeffprovides:\n  - capability: auth.oidc\n  - capability: auth.session\n",
		"followed by key":  "provides:\n  - capability: auth.oidc\n  - capability: auth.session\nconsumes: []\n",
	}
	for name, src := range cases {
		m, warn := parseManifest([]byte(src))
		if !reflect.DeepEqual(caps(m), want) {
			t.Errorf("%s: provides = %v, want %v", name, caps(m), want)
		}
		if len(warn) != 0 {
			t.Errorf("%s: unexpected warnings %v", name, warn)
		}
	}
}

func TestParseManifest_ConsumesAndNextKeyAfterList(t *testing.T) {
	m, _ := parseManifest([]byte("id: x\nprovides: [a]\nconsumes:\n  - b\nsummary: s\n"))
	if len(m.Consumes) != 1 || m.Consumes[0].Capability != "b" || m.Summary != "s" || m.ID != "x" {
		t.Fatalf("unexpected: %+v", m)
	}
}

func TestParseManifest_NothingIsDroppedSilently(t *testing.T) {
	for name, src := range map[string]string{
		"entry without capability": "provides:\n  - description: login\n",
		"flow map without key":     "provides: [{name: x}]\n",
		"not a list":               "provides: auth.oidc\n",
	} {
		m, warn := parseManifest([]byte(src))
		if len(m.Provides) != 0 && name != "not a list" {
			t.Errorf("%s: got %v", name, caps(m))
		}
		if len(warn) == 0 {
			t.Errorf("%s: expected a warning", name)
		}
	}
	for _, src := range []string{"provides: []\n", "provides:\n", "consumes: []   # none\n"} {
		if _, warn := parseManifest([]byte(src)); len(warn) != 0 {
			t.Errorf("%q: empty list must not warn, got %v", src, warn)
		}
	}
}

func TestParseManifest_BOMAndBrokenID(t *testing.T) {
	if m, _ := parseManifest([]byte("\ufeffid: widgets\n")); m.ID != "widgets" {
		t.Errorf("BOM dropped the id: %q", m.ID)
	}
	m, warn := parseManifest([]byte("id: [unclosed\n"))
	if m.ID != "" || len(warn) == 0 || !strings.Contains(warn[0], "id") {
		t.Errorf("unclosed id: %q, %v", m.ID, warn)
	}
}
