package scan

import (
	"strings"
	"testing"
)

func findLink(ls []Link, from, to string) *Link {
	for i := range ls {
		if ls[i].From == from && ls[i].To == to {
			return &ls[i]
		}
	}
	return nil
}

func TestOwnTaskAndNotesFilesAreNotWeakEvidence(t *testing.T) {
	root := t.TempDir()
	mkrepo(t, root, "tool-a", "", map[string]string{
		".relai/notes.md":      "tested on shop-front",
		".beads/issues.jsonl":  "shop-front",
		"openspec/specs/x.md":  "shop-front",
		"TODO.md":              "shop-front",
		"docs/00-handoff.md":   "shop-front",
		"docs/00-passation.md": "shop-front",
		"sub/tasks.md":         "shop-front",
	})
	mkrepo(t, root, "tool-b", "", map[string]string{"src/app.go": "// calls shop-front\n"})
	mkrepo(t, root, "shop-front", "", map[string]string{})
	repos, _ := Find(root)
	if l := findLink(Links(repos), "tool-a", "shop-front"); l != nil {
		t.Errorf("own task/notes/spec/handoff files must not make a weak mention: %+v", l)
	}
	if l := findLink(LinksWith(repos, Options{}), "tool-a", "shop-front"); l == nil || l.Weak != 7 {
		t.Errorf("without the rule all 7 files count: %+v", l)
	}
	if l := findLink(Links(repos), "tool-b", "shop-front"); l == nil || l.Weak != 1 {
		t.Errorf("a source file still counts: %+v", l)
	}
}

func TestOrdinaryWordNameNeedsAPathLikeContext(t *testing.T) {
	for name, want := range map[string]bool{"portfolio": true, "webshop": true, "web-shop": false, "Portfolio": false, "api2": false, "shop_x": false, "a.b": false, "averyveryverylongname": false} {
		if got := isOrdinaryWord(name, DefaultWordMaxLen); got != want {
			t.Errorf("isOrdinaryWord(%q) = %v, want %v", name, got, want)
		}
	}
	root := t.TempDir()
	mkrepo(t, root, "user", "", map[string]string{
		"prose.md":  "The portfolio is nice.\n",
		"path.sh":   "ls portfolio/src\n",
		"url.txt":   "https://github.com/acme/portfolio\n",
		"clone.sh":  "git clone x/y/portfolio.git\n",
		"import.go": "import \"example.com/acme/portfolio\"\n",
	})
	mkrepo(t, root, "portfolio", "", map[string]string{})
	repos, _ := Find(root)
	if l := findLink(Links(repos), "user", "portfolio"); l == nil || l.Weak != 4 {
		t.Errorf("want 4 files (path, url, .git, import) and not the prose one: %+v", l)
	}
	if l := findLink(LinksWith(repos, Options{}), "user", "portfolio"); l == nil || l.Weak != 5 {
		t.Errorf("with the rule off the prose file counts too: %+v", l)
	}
}

func TestDistinctiveNameStaysABareWordMention(t *testing.T) {
	root := t.TempDir()
	mkrepo(t, root, "user", "", map[string]string{"prose.md": "We deploy shop-front nightly.\n"})
	mkrepo(t, root, "shop-front", "", map[string]string{})
	repos, _ := Find(root)
	if l := findLink(Links(repos), "user", "shop-front"); l == nil || l.Weak != 1 {
		t.Errorf("a name with a dash is not an ordinary word: %+v", l)
	}
}

func TestStrongMentionIsNeverDroppedByTheWeakRules(t *testing.T) {
	root := t.TempDir()
	mkrepo(t, root, "user", "", map[string]string{
		"TODO.md":         "see https://github.com/acme/portfolio\n",
		".beads/x.jsonl":  "https://github.com/acme/portfolio",
		"deploy/ref.sh":   "cd ../portfolio\n",
		".relai/notes.md": "../portfolio",
	})
	mkrepo(t, root, "portfolio", "https://github.com/acme/portfolio", map[string]string{})
	repos, _ := Find(root)
	with, without := findLink(Links(repos), "user", "portfolio"), findLink(LinksWith(repos, Options{}), "user", "portfolio")
	if with == nil || without == nil || with.Strong != without.Strong || with.Strong < 3 {
		t.Errorf("strong evidence must be identical with and without the rules: %+v vs %+v", with, without)
	}
}

func TestProposeUsesTheIDWhenThereIsOneAndMarksFolderNames(t *testing.T) {
	root := t.TempDir()
	mkrepo(t, root, "user", "", map[string]string{"relai.yaml": "id: user\nsummary: x\n", "a.sh": "cd ../with-id\ncd ../no-id\n"})
	mkrepo(t, root, "with-id", "", map[string]string{"relai.yaml": "id: the-id\nsummary: x\n"})
	mkrepo(t, root, "no-id", "", map[string]string{})
	repos, _ := Find(root)
	prop := Propose(repos, Links(repos), 0)
	if !strings.Contains(prop, "  - the-id   # strong: 1 files\n") || !strings.Contains(prop, "  - no-id   # strong: 1 files; folder name, no relai.yaml yet\n") {
		t.Errorf("propose:\n%s", prop)
	}
}
