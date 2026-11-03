package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCopy_FreshTarget(t *testing.T) {
	root := t.TempDir()
	tpl := filepath.Join(root, "tpl")
	writeFile(t, filepath.Join(tpl, "README.md"), "hello")
	writeFile(t, filepath.Join(tpl, "sub", "f.txt"), "nested")
	writeFile(t, filepath.Join(tpl, ".git", "HEAD"), "ref: refs/heads/main")

	dir := filepath.Join(root, "new-repo")
	if err := Copy(tpl, dir); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("README.md = %q, %v", b, err)
	}
	b, err = os.ReadFile(filepath.Join(dir, "sub", "f.txt"))
	if err != nil || string(b) != "nested" {
		t.Fatalf("sub/f.txt = %q, %v", b, err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".git")); !os.IsNotExist(err) {
		t.Errorf(".git should not have been copied, stat err = %v", err)
	}
}

func TestCopy_RefusesNonEmptyTarget(t *testing.T) {
	root := t.TempDir()
	tpl := filepath.Join(root, "tpl")
	writeFile(t, filepath.Join(tpl, "a.txt"), "x")

	dir := filepath.Join(root, "existing")
	writeFile(t, filepath.Join(dir, "already-here.txt"), "y")

	if err := Copy(tpl, dir); err == nil {
		t.Fatal("expected an error scaffolding into a non-empty directory")
	}
	if _, err := os.Stat(filepath.Join(dir, "a.txt")); !os.IsNotExist(err) {
		t.Errorf("template file should not have been copied into the existing dir")
	}
}

func TestSubstitute(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "relai.yaml"), "id: {{.RepoName}}\nsummary: \"\"\n")
	writeFile(t, filepath.Join(dir, "plain.txt"), "no tokens here")
	writeFile(t, filepath.Join(dir, "unresolved.txt"), "{{.NotAField}}")
	// A byte sequence that happens to contain "{{" but isn't valid template syntax.
	writeFile(t, filepath.Join(dir, "binaryish.bin"), "\x89PNG\r\n{{\x1a\n\x00\x00\x00")

	if err := Substitute(dir, Tokens{RepoName: "widgets"}); err != nil {
		t.Fatal(err)
	}

	b, _ := os.ReadFile(filepath.Join(dir, "relai.yaml"))
	if string(b) != "id: widgets\nsummary: \"\"\n" {
		t.Errorf("relai.yaml = %q", b)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "plain.txt"))
	if string(b) != "no tokens here" {
		t.Errorf("plain.txt = %q", b)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "unresolved.txt"))
	if string(b) != "{{.NotAField}}" {
		t.Errorf("unresolved.txt should be left unchanged, got %q", b)
	}
	b, _ = os.ReadFile(filepath.Join(dir, "binaryish.bin"))
	if string(b) != "\x89PNG\r\n{{\x1a\n\x00\x00\x00" {
		t.Errorf("binaryish.bin should be left unchanged, got %q", b)
	}
}

func TestNew_FillsMissingContractWithoutOverwritingTemplate(t *testing.T) {
	root := t.TempDir()
	tpl := filepath.Join(root, "tpl")
	writeFile(t, filepath.Join(tpl, "AGENTS.md"), "# {{.RepoName}} — custom agents file\n")
	// No relai.yaml in the template: initrepo.Init should write one.

	dir := filepath.Join(root, "widgets")
	if _, err := New(tpl, dir, Tokens{RepoName: "widgets"}); err != nil {
		t.Fatal(err)
	}

	b, err := os.ReadFile(filepath.Join(dir, "AGENTS.md"))
	if err != nil || string(b) != "# widgets — custom agents file\n" {
		t.Fatalf("AGENTS.md = %q, %v (template-provided file must not be overwritten)", b, err)
	}
	if _, err := os.Stat(filepath.Join(dir, "relai.yaml")); err != nil {
		t.Errorf("relai.yaml should have been written by initrepo.Init: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".relai", "notes.md")); err != nil {
		t.Errorf(".relai/notes.md should have been written by initrepo.Init: %v", err)
	}
}

func TestNew_MissingTemplateSource(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "widgets")

	_, err := New(filepath.Join(root, "does-not-exist"), dir, Tokens{RepoName: "widgets"})
	if err == nil {
		t.Fatal("expected an error for a missing template source")
	}
	if got := err.Error(); !strings.Contains(got, "not found") {
		t.Errorf("error should clearly name the problem, got %q", got)
	}
	if _, statErr := os.Stat(dir); !os.IsNotExist(statErr) {
		t.Errorf("nothing should have been scaffolded on a missing template source")
	}
}

func TestNew_RefusesNonEmptyTarget(t *testing.T) {
	root := t.TempDir()
	tpl := filepath.Join(root, "tpl")
	writeFile(t, filepath.Join(tpl, "a.txt"), "x")

	dir := filepath.Join(root, "existing")
	writeFile(t, filepath.Join(dir, "already-here.txt"), "y")

	if _, err := New(tpl, dir, Tokens{RepoName: "x"}); err == nil {
		t.Fatal("expected an error scaffolding into a non-empty directory")
	}
}
