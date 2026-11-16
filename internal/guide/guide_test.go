package guide

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestInstall(t *testing.T) {
	t.Run("fresh then up to date", func(t *testing.T) {
		home := t.TempDir()
		rs, err := Install(home, false)
		if err != nil {
			t.Fatal(err)
		}
		for i, rel := range Targets {
			if rs[i].Status != Written || read(t, filepath.Join(home, rel)) != Skill() {
				t.Errorf("%s: %+v", rel, rs[i])
			}
		}
		rs, err = Install(home, false)
		if err != nil || rs[0].Status != UpToDate || rs[1].Status != UpToDate {
			t.Errorf("second run: %+v %v", rs, err)
		}
	})
	t.Run("differs without force writes nothing", func(t *testing.T) {
		home := t.TempDir()
		other := filepath.Join(home, Targets[1])
		os.MkdirAll(filepath.Dir(other), 0o755)
		os.WriteFile(other, []byte("mine"), 0o644)
		rs, err := Install(home, false)
		if err == nil || rs[1].Status != Differs {
			t.Fatalf("want error and Differs, got %+v %v", rs, err)
		}
		if _, e := os.Stat(filepath.Join(home, Targets[0])); !os.IsNotExist(e) {
			t.Error("first target was written")
		}
		if read(t, other) != "mine" {
			t.Error("differing file changed")
		}
	})
	t.Run("force overwrites", func(t *testing.T) {
		home := t.TempDir()
		other := filepath.Join(home, Targets[1])
		os.MkdirAll(filepath.Dir(other), 0o755)
		os.WriteFile(other, []byte("mine"), 0o644)
		if _, err := Install(home, true); err != nil {
			t.Fatal(err)
		}
		if read(t, other) != Skill() || read(t, filepath.Join(home, Targets[0])) != Skill() {
			t.Error("not overwritten")
		}
	})
	t.Run("symlinked file refused even with force", func(t *testing.T) {
		home := t.TempDir()
		dst := filepath.Join(home, "elsewhere")
		os.WriteFile(dst, []byte("x"), 0o644)
		link := filepath.Join(home, Targets[0])
		os.MkdirAll(filepath.Dir(link), 0o755)
		if err := os.Symlink(dst, link); err != nil {
			t.Skip(err)
		}
		if _, err := Install(home, true); err == nil {
			t.Fatal("want error")
		}
		if read(t, dst) != "x" {
			t.Error("followed the link")
		}
		if _, e := os.Stat(filepath.Join(home, Targets[1])); !os.IsNotExist(e) {
			t.Error("other target was written")
		}
	})
	t.Run("symlinked directory refused", func(t *testing.T) {
		home := t.TempDir()
		real := filepath.Join(home, "real")
		os.MkdirAll(real, 0o755)
		link := filepath.Join(home, filepath.Dir(Targets[1]))
		os.MkdirAll(filepath.Dir(link), 0o755)
		if err := os.Symlink(real, link); err != nil {
			t.Skip(err)
		}
		if _, err := Install(home, true); err == nil {
			t.Fatal("want error")
		}
		if es, _ := os.ReadDir(real); len(es) != 0 {
			t.Error("wrote through the link")
		}
	})
}

func TestSkillFrontmatter(t *testing.T) {
	s := Skill()
	if !strings.HasPrefix(s, "---\nname: relai\ndescription: ") || !strings.HasSuffix(s, Text()) {
		t.Errorf("bad skill: %q", s[:60])
	}
	if n := strings.Count(Text(), "\n"); n > 50 {
		t.Errorf("guide has %d lines, want at most 50", n)
	}
}

// Every `relai <word>` in the guide must be a subcommand main.go handles.
func TestGuideMentionsOnlyKnownSubcommands(t *testing.T) {
	known := map[string]bool{"init": true, "handoff": true, "discover": true, "index": true, "find": true,
		"note": true, "validate": true, "tidy": true, "workspace": true, "new": true, "guide": true, "version": true}
	main := read(t, filepath.Join("..", "..", "cmd", "relai", "main.go"))
	re := regexp.MustCompile("`relai ([a-z]+)")
	for _, m := range re.FindAllStringSubmatch(Text(), -1) {
		if !known[m[1]] || !strings.Contains(main, `case "`+m[1]+`"`) {
			t.Errorf("guide mentions `relai %s`, which main.go does not handle", m[1])
		}
	}
}
