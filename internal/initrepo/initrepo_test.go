package initrepo

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitIsIdempotentAndNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	r, err := Init(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Created) != 3 {
		t.Fatalf("want 3 files created, got %v", r.Created)
	}
	custom := []byte("# my own notes\n")
	notes := filepath.Join(dir, ".relai", "notes.md")
	_ = os.WriteFile(notes, custom, 0o644)

	r, err = Init(dir, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Created) != 0 || len(r.Skipped) != 3 {
		t.Fatalf("second run must only skip, got %+v", r)
	}
	if got, _ := os.ReadFile(notes); string(got) != string(custom) {
		t.Fatal("existing notes.md was overwritten")
	}
}

func TestInitLinksExistingClaudeMd(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte("# rules\n"), 0o644)
	if _, err := Init(dir, true); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(dir, "AGENTS.md"))
	if err != nil || target != "CLAUDE.md" {
		t.Fatalf("AGENTS.md should link to CLAUDE.md, got %q (%v)", target, err)
	}
}
