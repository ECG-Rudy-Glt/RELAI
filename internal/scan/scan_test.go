package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// mkrepo creates a git repo with the given files and, optionally, an origin remote.
func mkrepo(t *testing.T, root, name, remote string, files map[string]string) {
	t.Helper()
	dir := filepath.Join(root, name)
	_ = os.MkdirAll(dir, 0o755)
	run := func(args ...string) {
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	run("init", "-q", "-b", "main")
	if remote != "" {
		run("remote", "add", "origin", remote)
	}
	files["README.md"] = "# " + name + "\n"
	for p, body := range files {
		_ = os.MkdirAll(filepath.Dir(filepath.Join(dir, p)), 0o755)
		_ = os.WriteFile(filepath.Join(dir, p), []byte(body), 0o644)
	}
	run("add", "-A")
	run("commit", "-q", "-m", "x")
}

func cluster(t *testing.T) string {
	root := t.TempDir()
	mkrepo(t, root, "homelab", "git@github.com:acme/homelab.git", map[string]string{
		"docs/sites.md":  "The portfolio site is deployed from here. See ../portfolio/ for its source.\nAlso https://github.com/acme/shelfmark.git is cloned by the playbook.\n",
		"ansible/x.yml":  "# deploys portfolio and shelfmark\n",
		"relai.yaml":     "id: homelab\nsummary: infra\n",
		".beads/x":       "tracker",
		"graphify-out/g": "portfolio portfolio portfolio",
	})
	mkrepo(t, root, "portfolio", "https://github.com/acme/portfolio", map[string]string{"NOTES.md": "Hosted on the homelab, nothing else.\n"})
	mkrepo(t, root, "shelfmark", "git@github.com:acme/shelfmark.git", map[string]string{"a.txt": "standalone\n"})
	mkrepo(t, root, "ring", "", map[string]string{"b.txt": "the ring of fire\n"}) // short, common name
	_ = os.MkdirAll(filepath.Join(root, "not-a-repo"), 0o755)
	_ = os.MkdirAll(filepath.Join(root, ".hidden", ".git"), 0o755)
	return root
}

func TestFindListsRepositoriesAndWhatTheyHave(t *testing.T) {
	root := cluster(t)
	repos, err := Find(root)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, r := range repos {
		names = append(names, r.Name)
	}
	if strings.Join(names, ",") != "homelab,portfolio,ring,shelfmark" {
		t.Fatalf("repos = %v (a plain folder and a hidden one must not be listed)", names)
	}
	h := repos[0]
	if !h.Relai || !h.Tracker || h.ID != "homelab" || h.Slug != "acme/homelab" || h.LastCommit == "" {
		t.Errorf("homelab = %+v", h)
	}
	if repos[1].Relai || repos[1].Slug != "acme/portfolio" {
		t.Errorf("portfolio = %+v (https remote slug, no relai.yaml)", repos[1])
	}
}

func TestLinksSeparateStrongFromWeakAndSkipGeneratedFiles(t *testing.T) {
	root := cluster(t)
	repos, _ := Find(root)
	ls := Links(repos)
	get := func(from, to string) *Link {
		for i := range ls {
			if ls[i].From == from && ls[i].To == to {
				return &ls[i]
			}
		}
		return nil
	}
	hp := get("homelab", "portfolio")
	if hp == nil || hp.Strong != 1 || len(hp.Kinds) != 1 || hp.Kinds[0] != "relative path" {
		t.Fatalf("homelab -> portfolio = %+v, want strong via a relative path", hp)
	}
	if hp.Weak != 2 { // docs/sites.md and ansible/x.yml; graphify-out is generated and excluded
		t.Errorf("weak files = %d, want 2 (graphify-out must be ignored)", hp.Weak)
	}
	hs := get("homelab", "shelfmark")
	if hs == nil || hs.Strong != 1 || hs.Kinds[0] != "remote URL" {
		t.Errorf("homelab -> shelfmark = %+v, want strong via the remote URL", hs)
	}
	if get("homelab", "ring") != nil {
		t.Errorf("a 4-letter common word must not be searched as a weak mention unless it is long enough and present")
	}
	if l := get("portfolio", "homelab"); l == nil || l.Strong != 0 || l.Weak != 1 {
		t.Errorf("portfolio -> homelab = %+v, want one weak mention (the name as a word, no URL)", l)
	}
	if get("portfolio", "portfolio") != nil || get("shelfmark", "homelab") != nil {
		t.Error("self links or invented links")
	}
	if ls[0].Strong < ls[len(ls)-1].Strong {
		t.Error("links must be ordered strongest first")
	}
	for _, l := range ls {
		for _, e := range l.Evidence {
			if strings.Contains(e, "standalone") {
				t.Errorf("evidence must be paths, never content: %q", e)
			}
		}
	}
}

func TestTextAndProposeSayWhatTheyAreAndWriteNothing(t *testing.T) {
	root := cluster(t)
	repos, _ := Find(root)
	ls := Links(repos)
	txt := Text(root, repos, ls)
	if !strings.Contains(txt, "MENTIONS") || !strings.Contains(txt, "not verified") || !strings.Contains(txt, "homelab -> portfolio") {
		t.Errorf("text:\n%s", txt)
	}
	before, _ := os.ReadFile(filepath.Join(root, "homelab", "relai.yaml"))
	prop := Propose(repos, ls, 0)
	after, _ := os.ReadFile(filepath.Join(root, "homelab", "relai.yaml"))
	if string(before) != string(after) {
		t.Error("Propose wrote to a relai.yaml")
	}
	if !strings.Contains(prop, "nothing was written") || !strings.Contains(prop, "# homelab/relai.yaml\nlinks:\n  - portfolio   # strong: 1 files") {
		t.Errorf("propose:\n%s", prop)
	}
	if strings.Contains(prop, "# ring/relai.yaml") {
		t.Errorf("a repository with no evidence must get no proposal:\n%s", prop)
	}
}

func TestSlugForms(t *testing.T) {
	for in, want := range map[string]string{
		"git@github.com:acme/app.git": "acme/app", "https://github.com/acme/app": "acme/app", "https://gitlab.com/g/sub/app.git": "sub/app",
		"ssh://git@host:2222/acme/app.git": "acme/app", "": "", "nonsense": "",
	} {
		if got := slugOf(in); got != want {
			t.Errorf("slugOf(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFindFollowsSymbolicLinksToRepositoriesAndListsEachOnce(t *testing.T) {
	root := t.TempDir()
	elsewhere := filepath.Join(root, "projets")
	mkrepo(t, elsewhere, "real-app", "", map[string]string{})
	folder := filepath.Join(root, "home")
	_ = os.MkdirAll(folder, 0o755)
	if err := os.Symlink(filepath.Join(elsewhere, "real-app"), filepath.Join(folder, "app")); err != nil {
		t.Skip("no symlinks")
	}
	_ = os.Symlink(filepath.Join(elsewhere, "real-app"), filepath.Join(folder, "app-again")) // same repository, second name
	_ = os.Symlink(filepath.Join(root, "missing"), filepath.Join(folder, "broken"))
	repos, err := Find(folder)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "app" {
		t.Errorf("repos = %+v, want the linked repository once, under its first name, and no broken link", repos)
	}
}

func TestPromptTellsTheAgentToVerifyAndAskBeforeWriting(t *testing.T) {
	root := cluster(t)
	repos, _ := Find(root)
	p := Prompt(root, repos, Links(repos))
	for _, want := range []string{"MENTIONS", "not a verified link", "only then write", "Never copy secrets", "homelab -> portfolio", "portfolio — no relai.yaml (run relai init)"} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt lacks %q:\n%s", want, p)
		}
	}
}

func TestFindDepthLooksInsideFoldersButNeverInsideRepositories(t *testing.T) {
	root := t.TempDir()
	mkrepo(t, root, "top", "", map[string]string{"nested/deep/x.txt": "inside a repo: must not be searched for repos"})
	mkrepo(t, filepath.Join(root, "clients"), "acme", "", map[string]string{})
	mkrepo(t, filepath.Join(root, "clients"), "globex", "", map[string]string{})
	mkrepo(t, filepath.Join(root, "personal"), "acme", "", map[string]string{}) // same folder name, other place
	mkrepo(t, filepath.Join(root, "node_modules"), "dep", "", map[string]string{})
	names := func(depth int) string {
		repos, err := FindDepth(root, depth)
		if err != nil {
			t.Fatal(err)
		}
		var n []string
		for _, r := range repos {
			n = append(n, r.Name)
		}
		return strings.Join(n, ",")
	}
	if got := names(1); got != "top" {
		t.Errorf("depth 1 = %q, want only the repo directly inside", got)
	}
	if got := names(2); got != "clients/acme,clients/globex,personal/acme,top" {
		t.Errorf("depth 2 = %q: nested repos must be named by path so two 'acme' stay distinct, node_modules skipped", got)
	}
}

func TestWeakMentionsInConfigFilesCountMoreThanInDocuments(t *testing.T) {
	root := t.TempDir()
	mkrepo(t, root, "deployer", "", map[string]string{
		".github/workflows/deploy.yml": "run: deploy webshop\n",
		"k8s/webshop.yaml":             "name: webshop\n",
		"NOTES.md":                     "webshop webshop\n",
	})
	mkrepo(t, root, "chatty", "", map[string]string{"a.md": "webshop", "b.md": "webshop", "c.md": "webshop"})
	mkrepo(t, root, "webshop", "", map[string]string{})
	repos, _ := Find(root)
	ls := Links(repos)
	var dep, chat *Link
	for i := range ls {
		if ls[i].To == "webshop" && ls[i].From == "deployer" {
			dep = &ls[i]
		}
		if ls[i].To == "webshop" && ls[i].From == "chatty" {
			chat = &ls[i]
		}
	}
	if dep == nil || dep.WeakConfig != 2 || dep.Weak != 3 {
		t.Fatalf("deployer -> webshop = %+v, want 3 weak files of which 2 in CI/deploy config", dep)
	}
	if chat == nil || chat.WeakConfig != 0 || chat.Weak != 3 {
		t.Fatalf("chatty -> webshop = %+v, want 3 weak files, none in config", chat)
	}
	prop := Propose(repos, ls, 0) // default threshold of 5 documents
	if !strings.Contains(prop, "# deployer/relai.yaml") || strings.Contains(prop, "# chatty/relai.yaml") {
		t.Errorf("two config files should be proposed, three documents should not:\n%s", prop)
	}
	if !strings.Contains(Propose(repos, ls, 3), "# chatty/relai.yaml") {
		t.Error("--min-weak 3 should make three documents enough")
	}
}

func TestWorktreesAreOneRepo(t *testing.T) {
	root := t.TempDir()
	main := filepath.Join(root, "app")
	run := func(dir string, args ...string) {
		t.Helper()
		c := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v %s", args, err, out)
		}
	}
	if err := os.MkdirAll(main, 0o755); err != nil {
		t.Fatal(err)
	}
	run(main, "init", "-q")
	run(main, "commit", "-q", "--allow-empty", "-m", "x")
	run(main, "worktree", "add", "-q", filepath.Join(root, "app-feature"), "-b", "feature")
	repos, err := Find(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 1 || repos[0].Name != "app" || len(repos[0].Worktrees) != 1 {
		t.Fatalf("want one repo app with one worktree, got %+v", repos)
	}
}
