package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

func findFixture() []index.Entry {
	cap := func(n string) []index.Capability { return []index.Capability{{Capability: n}} }
	return []index.Entry{
		{Source: "../auth", RelaiYAML: index.Manifest{ID: "auth", Summary: "login", Provides: cap("auth.oidc")}},
		{Source: "../web", RelaiYAML: index.Manifest{ID: "web", Summary: "site", Consumes: cap("auth.oidc")}},
	}
}

func TestWriteFindConsumers(t *testing.T) {
	var b bytes.Buffer
	if err := writeFind(&b, findFixture(), "auth.oidc", true, false, index.Manifest{}); err != nil {
		t.Fatal(err)
	}
	got := b.String()
	if !strings.HasPrefix(got, `repos in the index that consume "auth.oidc":`) || !strings.Contains(got, "- web (../web): site") || strings.Contains(got, "- auth") {
		t.Errorf("unexpected output:\n%s", got)
	}
}

func TestWriteFindConsumersNothing(t *testing.T) {
	var b bytes.Buffer
	_ = writeFind(&b, findFixture(), "nope", true, false, index.Manifest{})
	if !strings.HasPrefix(b.String(), `nothing in the index consumes "nope"`) {
		t.Errorf("unexpected output: %s", b.String())
	}
	b.Reset()
	_ = writeFind(&b, findFixture(), "nope", false, false, index.Manifest{})
	if b.String() != "nothing in the index provides \"nope\"\n" {
		t.Errorf("providers wording changed: %q", b.String())
	}
}

func TestWriteFindJSON(t *testing.T) {
	var b bytes.Buffer
	if err := writeFind(&b, findFixture(), "auth.oidc", true, true, index.Manifest{}); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Capability string
		Mode       string
		Matches    []index.Entry
	}
	if err := json.Unmarshal(b.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Mode != "consumers" || got.Capability != "auth.oidc" || len(got.Matches) != 1 || got.Matches[0].RelaiYAML.ID != "web" {
		t.Errorf("unexpected: %+v", got)
	}
	b.Reset()
	_ = writeFind(&b, findFixture(), "nope", false, true, index.Manifest{})
	if !strings.Contains(b.String(), `"matches": []`) || !strings.Contains(b.String(), `"mode": "providers"`) {
		t.Errorf("empty result must be []: %s", b.String())
	}
}
