package index

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// model is the shape the conformance cases use: capabilities and links as plain names, empty
// values left out.
func model(m Manifest) map[string]any {
	out := map[string]any{}
	str := func(k, v string) {
		if v != "" {
			out[k] = v
		}
	}
	list := func(k string, v []string) {
		if len(v) > 0 {
			l := make([]any, len(v))
			for i, x := range v {
				l[i] = x
			}
			out[k] = l
		}
	}
	names := func(c []Capability) []string {
		var n []string
		for _, x := range c {
			n = append(n, x.Capability)
		}
		return n
	}
	str("apiVersion", m.APIVersion)
	str("id", m.ID)
	str("summary", m.Summary)
	str("conventions", m.Conventions)
	str("tasks", m.Tasks)
	list("owners", m.Owners)
	list("provides", names(m.Provides))
	list("consumes", names(m.Consumes))
	list("links", m.Links)
	return out
}

// dropEmpty removes the keys a reader may print with an empty value.
func dropEmpty(m map[string]any) map[string]any {
	for k, v := range m {
		if l, ok := v.([]any); (ok && len(l) == 0) || v == "" || v == nil {
			delete(m, k)
		}
	}
	return m
}

// TestConformance checks both readers of relai.yaml, the Go one and the Python one written from
// the specification page alone, against spec/conformance.
func TestConformance(t *testing.T) {
	dir := filepath.Join("..", "..", "spec", "conformance")
	cases, _ := filepath.Glob(filepath.Join(dir, "*.yaml"))
	if len(cases) == 0 {
		t.Fatal("no conformance cases found")
	}
	python, _ := exec.LookPath("python3")
	if python == "" {
		t.Log("python3 not found: the Python reader is not checked")
	}
	for _, c := range cases {
		name := strings.TrimSuffix(filepath.Base(c), ".yaml")
		t.Run(name, func(t *testing.T) {
			src, err := os.ReadFile(c)
			if err != nil {
				t.Fatal(err)
			}
			raw, err := os.ReadFile(strings.TrimSuffix(c, ".yaml") + ".json")
			if err != nil {
				t.Fatal(err)
			}
			var want map[string]any
			if err := json.Unmarshal(raw, &want); err != nil {
				t.Fatal(err)
			}
			m, warn := ParseManifestWarn(src)
			if len(warn) > 0 {
				t.Errorf("Go reader warned on a valid file: %v", warn)
			}
			if got := model(m); !reflect.DeepEqual(got, want) {
				t.Errorf("Go reader:\n got %v\nwant %v", got, want)
			}
			if python == "" {
				return
			}
			out, err := exec.Command(python, filepath.Join("..", "..", "spec", "readers", "relai_yaml.py"), c).Output()
			if err != nil {
				t.Fatalf("Python reader failed: %v", err)
			}
			var got map[string]any
			if err := json.Unmarshal(out, &got); err != nil {
				t.Fatalf("Python reader printed something that is not JSON: %v", err)
			}
			if got = dropEmpty(got); !reflect.DeepEqual(got, want) {
				t.Errorf("Python reader:\n got %v\nwant %v", got, want)
			}
		})
	}
}
