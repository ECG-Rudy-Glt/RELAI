package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func isolateHub(t *testing.T) string {
	t.Helper()
	cache := t.TempDir()
	t.Setenv("RELAI_CACHE_DIR", cache)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
	return cache
}

func gitT(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func put(t *testing.T, root, rel, data string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func snap(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if strings.Contains(p, string(filepath.Separator)+".git"+string(filepath.Separator)) {
			return nil
		}
		b, _ := os.ReadFile(p)
		rel, _ := filepath.Rel(root, p)
		out[rel] = string(b)
		return nil
	})
	return out
}

// hubRepo creates a hub with `relai hub init`, committed and tagged v1.0.0.
func hubRepo(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "hub")
	var o, e bytes.Buffer
	if code := cmdHub([]string{"init", dir}, &o, &e); code != 0 {
		t.Fatalf("hub init: %d %s", code, e.String())
	}
	gitT(t, dir, "init", "-q")
	gitT(t, dir, "add", "-A")
	gitT(t, dir, "commit", "-q", "-m", "hub")
	gitT(t, dir, "tag", "v1.0.0")
	return dir
}

// follower creates a committed repository that pins hub at tag.
func follower(t *testing.T, hub, tag string) string {
	t.Helper()
	root := t.TempDir()
	put(t, root, "relai.yaml", "apiVersion: relai.dev/v0\nid: demo\nsummary: A demo repository.\nconventions: "+hub+"@"+tag+"\n")
	put(t, root, "AGENTS.md", "# demo\n\n## This repo\n- local rule\n")
	put(t, root, ".relai/notes.md", "# Notes\n")
	gitT(t, root, "init", "-q")
	gitT(t, root, "add", "-A")
	gitT(t, root, "commit", "-q", "-m", "init")
	return root
}

func run(f func([]string, io.Writer, io.Writer) int, args ...string) (int, string, string) {
	var o, e bytes.Buffer
	code := f(args, &o, &e)
	return code, o.String(), e.String()
}

func TestSyncCheck_EndToEnd(t *testing.T) {
	isolateHub(t)
	hub := hubRepo(t)
	root := follower(t, hub, "v1.0.0")

	before := snap(t, root)
	code, out, errOut := run(cmdSync, root)
	if code != 0 || !strings.Contains(out, "would write") || !strings.Contains(out, "Dry run") {
		t.Fatalf("dry run: %d\n%s%s", code, out, errOut)
	}
	if !reflect.DeepEqual(before, snap(t, root)) {
		t.Fatal("sync without --apply wrote something")
	}

	if code, out, e := run(cmdCheck, root); code != 1 || !strings.Contains(out, "not vendored yet") {
		t.Errorf("check before sync must fail: %d\n%s%s", code, out, e)
	}

	if code, out, e := run(cmdSync, "--apply", root); code != 0 || !strings.Contains(out, "wrote") {
		t.Fatalf("apply: %d\n%s%s", code, out, e)
	}
	applied := snap(t, root)
	if code, out, _ := run(cmdSync, "--apply", root); code != 0 || !strings.Contains(out, "up to date") || !reflect.DeepEqual(applied, snap(t, root)) {
		t.Errorf("second apply must change nothing:\n%s", out)
	}
	if code, out, e := run(cmdCheck, root); code != 0 {
		t.Errorf("check after sync: %d\n%s%s", code, out, e)
	}

	put(t, root, ".relai/conventions.md", "tampered\n")
	code, out, _ = run(cmdCheck, root)
	if code != 1 || !strings.Contains(out, ".relai/conventions.md: error: differs from the hub at v1.0.0") {
		t.Errorf("hand edit not detected: %d\n%s", code, out)
	}
	code, out, _ = run(cmdCheck, "--json", root)
	var fs []struct{ Level, File, Message string }
	if err := json.Unmarshal([]byte(out), &fs); err != nil || code != 1 {
		t.Fatalf("--json: %v %d\n%s", err, code, out)
	}
	found := false
	for _, f := range fs {
		found = found || (f.Level == "error" && f.File == ".relai/conventions.md")
	}
	if !found {
		t.Errorf("json findings: %v", fs)
	}
}

func TestCheck_OverridesAreInfoAndNeverFail(t *testing.T) {
	isolateHub(t)
	hub := hubRepo(t)
	root := follower(t, hub, "v1.0.0")
	if code, _, _ := run(cmdSync, "--apply", root); code != 0 {
		t.Fatal("sync")
	}
	put(t, root, "AGENTS.md", strings.Replace(readT(t, root, "AGENTS.md"), "- local rule\n", "- local rule\nOverride: no conventional commits here, the history is imported\n", 1))
	for _, args := range [][]string{{root}, {"--strict", root}} {
		code, out, e := run(cmdCheck, args...)
		if code != 0 || !strings.Contains(out, "info: declared override: Override: no conventional commits") {
			t.Errorf("%v: %d\n%s%s", args, code, out, e)
		}
	}
}

func readT(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestCheck_StrictTurnsWarningsIntoFailures(t *testing.T) {
	isolateHub(t)
	root := t.TempDir() // no AGENTS.md, no notes: warnings only
	put(t, root, "relai.yaml", "apiVersion: relai.dev/v0\nid: demo\nsummary: A demo repository.\n")
	gitT(t, root, "init", "-q")
	if code, out, _ := run(cmdCheck, root); code != 0 {
		t.Errorf("warnings alone must pass: %d\n%s", code, out)
	}
	if code, _, _ := run(cmdCheck, "--strict", root); code != 1 {
		t.Error("--strict must fail on a warning")
	}
}

func TestCheck_Outdated(t *testing.T) {
	isolateHub(t)
	hub := hubRepo(t)
	gitT(t, hub, "tag", "v1.9.0")
	gitT(t, hub, "tag", "v1.10.0")
	root := follower(t, hub, "v1.9.0")
	if code, out, e := run(cmdSync, "--apply", root); code != 0 {
		t.Fatalf("%s%s", out, e)
	}
	if _, out, _ := run(cmdCheck, root); strings.Contains(out, "newer version") {
		t.Errorf("without --outdated nothing is said about versions:\n%s", out)
	}
	code, out, _ := run(cmdCheck, "--outdated", root)
	if code != 0 || !strings.Contains(out, "warning: the hub has a newer version v1.10.0 (pinned: v1.9.0)") {
		t.Errorf("--outdated: %d\n%s", code, out)
	}
	if code, _, _ := run(cmdCheck, "--outdated", "--strict", root); code != 1 {
		t.Error("--outdated with --strict must fail")
	}
}

func TestSync_ClearErrors(t *testing.T) {
	isolateHub(t)
	hub := hubRepo(t)

	missing := follower(t, filepath.Join(t.TempDir(), "gone"), "v1")
	code, out, e := run(cmdSync, missing)
	if code != 1 || out != "" || !strings.HasPrefix(e, "relai: cannot read the hub ") || strings.Count(e, "\n") != 1 || strings.Contains(e, "fatal") {
		t.Errorf("missing hub: %d %q %q", code, out, e)
	}
	unknown := follower(t, hub, "v7.7.7")
	code, _, e = run(cmdSync, unknown)
	if code != 1 || !strings.Contains(e, `has no tag "v7.7.7"`) || strings.Count(e, "\n") != 1 || strings.Contains(e, "fatal") {
		t.Errorf("unknown tag: %d %q", code, e)
	}
	if code, out, _ := run(cmdCheck, unknown); code != 1 || !strings.Contains(out, `has no tag "v7.7.7"`) {
		t.Errorf("check with an unknown tag: %d\n%s", code, out)
	}
	plain := t.TempDir()
	put(t, plain, "relai.yaml", "id: x\n")
	if code, _, e := run(cmdSync, plain); code != 1 || !strings.Contains(e, "no `conventions:` line") {
		t.Errorf("no pin: %d %q", code, e)
	}
	dash := follower(t, "-evil", "v1")
	if code, _, e := run(cmdSync, dash); code != 1 || !strings.Contains(e, `must not start with "-"`) {
		t.Errorf("dash hub: %d %q", code, e)
	}
	dashTag := follower(t, hub, "-v1")
	if code, _, e := run(cmdSync, dashTag); code != 1 || !strings.Contains(e, `must not start with "-"`) {
		t.Errorf("dash tag: %d %q", code, e)
	}
}

func TestSync_CacheIsOutsideTheRepository(t *testing.T) {
	cache := isolateHub(t)
	hub := hubRepo(t)
	root := follower(t, hub, "v1.0.0")
	if code, _, e := run(cmdSync, "--apply", root); code != 0 {
		t.Fatal(e)
	}
	if entries, _ := os.ReadDir(filepath.Join(cache, "hubs")); len(entries) != 1 {
		t.Errorf("the hub is not in the cache directory: %v", entries)
	}
	for p := range snap(t, root) {
		if strings.Contains(p, "hubs") {
			t.Errorf("cache content in the repository: %s", p)
		}
	}
}

func TestHubInit_RefusesANonEmptyDirectory(t *testing.T) {
	isolateHub(t)
	dir := t.TempDir()
	put(t, dir, "keep.txt", "mine")
	code, _, e := run(cmdHub, "init", dir)
	if code != 1 || !strings.Contains(e, "not empty") {
		t.Errorf("%d %q", code, e)
	}
	if !reflect.DeepEqual(snap(t, dir), map[string]string{"keep.txt": "mine"}) {
		t.Error("a refused init wrote something")
	}
	if code, _, _ := run(cmdHub); code != 2 {
		t.Error("usage error expected")
	}
}
