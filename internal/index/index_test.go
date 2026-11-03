package index

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// gitRepo creates a git repo at dir with the given relai.yaml content (skipped if content == "").
func gitRepo(t *testing.T, dir, relaiYAML string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t.com",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t.com")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "--quiet", "--initial-branch=main")
	if relaiYAML != "" {
		if err := os.WriteFile(filepath.Join(dir, "relai.yaml"), []byte(relaiYAML), 0o644); err != nil {
			t.Fatal(err)
		}
	} else {
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("hi"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	run("add", "-A")
	run("commit", "--quiet", "-m", "init")
}

const sampleManifest = `apiVersion: relai.dev/v0
id: widgets
summary: "Makes widgets."
owners: [alice, bob]
provides:
  - capability: widgets.make
  - capability: widgets.paint
consumes: []           # - capability: auth.oidc
`

func TestBuild_ValidMissingAndEmptyManifest(t *testing.T) {
	root := t.TempDir()

	valid := filepath.Join(root, "valid")
	gitRepo(t, valid, sampleManifest)

	noManifest := filepath.Join(root, "no-manifest")
	gitRepo(t, noManifest, "")

	missing := filepath.Join(root, "does-not-exist")

	entries, skipped := Build([]string{valid, missing, noManifest})

	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1 (%+v)", len(entries), entries)
	}
	e := entries[0]
	if e.Source != valid {
		t.Errorf("source = %q, want %q", e.Source, valid)
	}
	if e.RelaiYAML.ID != "widgets" || e.RelaiYAML.Summary != "Makes widgets." {
		t.Errorf("manifest = %+v", e.RelaiYAML)
	}
	if len(e.RelaiYAML.Owners) != 2 || e.RelaiYAML.Owners[0] != "alice" || e.RelaiYAML.Owners[1] != "bob" {
		t.Errorf("owners = %v", e.RelaiYAML.Owners)
	}
	if len(e.RelaiYAML.Provides) != 2 ||
		e.RelaiYAML.Provides[0].Capability != "widgets.make" ||
		e.RelaiYAML.Provides[1].Capability != "widgets.paint" {
		t.Errorf("provides = %+v", e.RelaiYAML.Provides)
	}
	if len(e.RelaiYAML.Consumes) != 0 {
		t.Errorf("consumes = %+v, want empty", e.RelaiYAML.Consumes)
	}

	if len(skipped) != 2 {
		t.Fatalf("skipped = %d, want 2 (%+v)", len(skipped), skipped)
	}
	bySource := map[string]string{}
	for _, s := range skipped {
		bySource[s.Source] = s.Reason
	}
	if bySource[missing] == "" {
		t.Errorf("missing path not reported as skipped")
	}
	if bySource[noManifest] == "" {
		t.Errorf("no-manifest repo not reported as skipped")
	}
}

func TestBuild_CloneURL(t *testing.T) {
	root := t.TempDir()
	repoDir := filepath.Join(root, "cloneable")
	gitRepo(t, repoDir, sampleManifest)

	url := "file://" + repoDir
	entries, skipped := Build([]string{url})
	if len(skipped) != 0 {
		t.Fatalf("skipped = %+v, want none", skipped)
	}
	if len(entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries))
	}
	if entries[0].Source != url {
		t.Errorf("source = %q, want %q", entries[0].Source, url)
	}
	if entries[0].RelaiYAML.ID != "widgets" {
		t.Errorf("manifest = %+v", entries[0].RelaiYAML)
	}
}

func TestFind(t *testing.T) {
	entries := []Entry{
		{Source: "/repo-a", RelaiYAML: Manifest{ID: "a", Summary: "does a", Provides: []Capability{{Capability: "widgets.make"}}}},
		{Source: "/repo-b", RelaiYAML: Manifest{ID: "b", Summary: "does b", Provides: []Capability{{Capability: "widgets.paint"}}}},
	}

	found := Find(entries, "widgets.make")
	if len(found) != 1 || found[0].Source != "/repo-a" {
		t.Fatalf("Find(widgets.make) = %+v", found)
	}

	none := Find(entries, "widgets.ship")
	if len(none) != 0 {
		t.Fatalf("Find(widgets.ship) = %+v, want none", none)
	}
}

func TestReadList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "repos.txt")
	content := "\n# a comment\n/one\n  /two  \n\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadList(path)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/one", "/two"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("got %v, want %v", got, want)
	}
}
