package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// mkRepo creates a real git repository at dir/name with a relai.yaml (id = name), and returns its path.
func mkRepo(t *testing.T, dir, name string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", p, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git init: %v: %s", err, out)
	}
	yaml := "id: " + name + "\nsummary: test repo\n"
	if err := os.WriteFile(filepath.Join(p, "relai.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestScanSourcesAccumulates(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	ra := mkRepo(t, a, "one")
	rb := mkRepo(t, b, "two")
	var out bytes.Buffer
	got, err := scanSources([]string{a, b}, 1, &out)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0] != ra || got[1] != rb {
		t.Fatalf("want [%s %s], got %v", ra, rb, got)
	}
	if n := strings.Count(out.String(), "scan: "); n != 2 {
		t.Errorf("want one scan line per folder, got:\n%s", out.String())
	}
}

func TestScanSourcesOverlapNoDuplicate(t *testing.T) {
	root := t.TempDir()
	r := mkRepo(t, root, "one")
	mkRepo(t, root, "two")
	link := filepath.Join(t.TempDir(), "link")
	if err := os.Symlink(root, link); err != nil {
		t.Skip("no symlinks:", err)
	}
	// the same folder twice, a symlink to it, and a folder that is itself the repository
	got, err := scanSources([]string{root, root, link, r}, 1, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 repositories once each, got %v", got)
	}
}

func TestScanSourcesSingleKeepsOutput(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, root, "one")
	p := filepath.Join(root, "bare")
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
	_ = exec.Command("git", "-C", p, "init", "-q").Run()
	var out bytes.Buffer
	if _, err := scanSources([]string{root}, 1, &out); err != nil {
		t.Fatal(err)
	}
	want := "scan: 2 repositories in " + root + ", 1 with a relai.yaml; without one (skipped, run `relai init` there to include them): bare\n"
	if out.String() != want {
		t.Errorf("got %q\nwant %q", out.String(), want)
	}
}

func TestScanSourcesMissingFolder(t *testing.T) {
	if _, err := scanSources([]string{filepath.Join(t.TempDir(), "nope")}, 1, &bytes.Buffer{}); err == nil {
		t.Error("a missing folder must be an error")
	}
}

// runIndexBuild builds the real binary once per test and runs `relai index build` with args.
func runIndexBuild(t *testing.T, args ...string) (string, []string) {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "relai")
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	idx := filepath.Join(t.TempDir(), "index.json")
	cmd := exec.Command(bin, append([]string{"index", "build", "-o", idx}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("index build: %v\n%s", err, out)
	}
	b, err := os.ReadFile(idx)
	if err != nil {
		t.Fatal(err)
	}
	var entries []struct {
		Source string
	}
	if err := json.Unmarshal(b, &entries); err != nil {
		t.Fatal(err)
	}
	var sources []string
	for _, e := range entries {
		sources = append(sources, e.Source)
	}
	return string(out), sources
}

func TestIndexBuildRepeatedScan(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	mkRepo(t, a, "one")
	mkRepo(t, b, "two")
	_, sources := runIndexBuild(t, "--scan", a, "--scan", b)
	if len(sources) != 2 {
		t.Fatalf("both folders must be indexed, got %v", sources)
	}
	// overlapping folders (root holds a and b at depth 2) must not duplicate
	_, sources = runIndexBuild(t, "--scan", a, "--scan", root, "--depth", "2")
	if len(sources) != 2 {
		t.Fatalf("overlap must not duplicate, got %v", sources)
	}
}

func TestIndexBuildSingleScanUnchanged(t *testing.T) {
	root := t.TempDir()
	mkRepo(t, root, "one")
	out, sources := runIndexBuild(t, "--scan", root)
	if len(sources) != 1 {
		t.Fatalf("got %v", sources)
	}
	if !strings.Contains(out, "scan: 1 repositories in "+root+", 1 with a relai.yaml\n") {
		t.Errorf("scan line changed:\n%s", out)
	}
}

func TestIndexBuildScansWithListFile(t *testing.T) {
	root := t.TempDir()
	a, b := filepath.Join(root, "a"), filepath.Join(root, "b")
	mkRepo(t, a, "one")
	mkRepo(t, b, "two")
	listed := mkRepo(t, filepath.Join(root, "c"), "three")
	list := filepath.Join(root, "list.txt")
	if err := os.WriteFile(list, []byte(listed+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, sources := runIndexBuild(t, "--scan", a, "--scan", b, list)
	if len(sources) != 3 {
		t.Fatalf("list file and both scans must add up, got %v", sources)
	}
}
