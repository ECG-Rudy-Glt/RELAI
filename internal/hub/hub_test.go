package hub

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/ECG-Rudy-Glt/RELAI/internal/validate"
)

// isolate keeps every git call of a test away from the real home and the real git configuration.
func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_SYSTEM", os.DevNull)
	t.Setenv("GIT_AUTHOR_NAME", "t")
	t.Setenv("GIT_AUTHOR_EMAIL", "t@t")
	t.Setenv("GIT_COMMITTER_NAME", "t")
	t.Setenv("GIT_COMMITTER_EMAIL", "t@t")
}

func gitIn(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func write(t *testing.T, root, rel, data string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func exists(root, rel string) bool {
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

// newHub makes a local git repository holding files, commits it and tags it tag.
func newHub(t *testing.T, tag string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	gitIn(t, dir, "init", "-q")
	commitHub(t, dir, tag, files, nil)
	return dir
}

// commitHub replaces the working tree of dir with files (and links), commits and tags it.
func commitHub(t *testing.T, dir, tag string, files map[string]string, links map[string]string) {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != ".git" {
			_ = os.RemoveAll(filepath.Join(dir, e.Name()))
		}
	}
	for rel, data := range files {
		write(t, dir, rel, data)
	}
	for rel, target := range links {
		if err := os.MkdirAll(filepath.Dir(filepath.Join(dir, rel)), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(target, filepath.Join(dir, rel)); err != nil {
			t.Fatal(err)
		}
	}
	gitIn(t, dir, "add", "-A")
	gitIn(t, dir, "commit", "-q", "--allow-empty", "-m", "hub "+tag)
	gitIn(t, dir, "tag", "-f", tag)
}

var baseHub = map[string]string{
	"CONVENTIONS.md":            "# Conventions\n\n- be kind\n",
	"skills/alpha/SKILL.md":     "alpha v1\n",
	"skills/alpha/extra/ref.md": "ref\n",
	"skills/beta/SKILL.md":      "beta v1\n",
	"ci/github.yml":             "name: relai\n",
	"ci/forgejo.yml":            "name: relai-forgejo\n",
	"ci/gitlab.yml":             "relai:\n  script: relai check\n",
}

const agentsText = "# repo\n\nIntro line.\n\n## Rules\n- local rule\n"

// newRepo makes the repository that follows the hub: relai.yaml and AGENTS.md.
func newRepo(t *testing.T, agents string) string {
	t.Helper()
	root := t.TempDir()
	write(t, root, "relai.yaml", "apiVersion: relai.dev/v0\nid: demo\nsummary: Demo.\n")
	if agents != "" {
		write(t, root, "AGENTS.md", agents)
	}
	return root
}

// snapshot describes every entry under root (content, or link target), to prove nothing changed.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			l, _ := os.Readlink(p)
			out[rel] = "link->" + l
		case d.IsDir():
			out[rel] = "dir"
		default:
			b, _ := os.ReadFile(p)
			out[rel] = string(b)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// sync fetches, loads, plans and (when apply) applies, the way `relai sync` does.
func syncOnce(t *testing.T, root string, pin Pin, cache string, apply bool) *Plan {
	t.Helper()
	dir, err := Fetch(pin, cache, false)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	p, err := BuildPlan(root, pin, c)
	if err != nil {
		t.Fatal(err)
	}
	if apply {
		if _, err := Apply(root, p); err != nil {
			t.Fatal(err)
		}
	}
	return p
}

func driftFiles(f []validate.Finding) []string {
	var out []string
	for _, x := range f {
		if x.Level == validate.Error {
			out = append(out, x.File)
		}
	}
	sort.Strings(out)
	return out
}

func TestParsePin(t *testing.T) {
	ok := map[string]Pin{
		"https://x.org/team/hub.git@v1.2.0": {"https://x.org/team/hub.git", "v1.2.0"},
		"git@host:team/hub.git@v3":          {"git@host:team/hub.git", "v3"},
		"/srv/hub@v1":                       {"/srv/hub", "v1"},
	}
	for in, want := range ok {
		got, err := ParsePin(in)
		if err != nil || got != want {
			t.Errorf("ParsePin(%q) = %v, %v; want %v", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "no-tag", "@v1", "hub@", "-hub@v1", "hub@-v1", "hub@--upload-pack=x", "a@b\x1b[2J@v1", "h-->x@v1"} {
		if _, err := ParsePin(bad); err == nil {
			t.Errorf("ParsePin(%q) must fail", bad)
		}
	}
}

func TestFetch_ClearErrors(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)

	_, err := Fetch(Pin{filepath.Join(t.TempDir(), "nope"), "v1"}, cache, false)
	if err == nil || !strings.Contains(err.Error(), "cannot read the hub") || strings.Contains(err.Error(), "fatal") || strings.Contains(err.Error(), "\n") {
		t.Errorf("missing hub: %v", err)
	}
	_, err = Fetch(Pin{h, "v9.9.9"}, cache, false)
	if err == nil || !strings.Contains(err.Error(), `has no tag "v9.9.9"`) || strings.Contains(err.Error(), "fatal") || strings.Contains(err.Error(), "\n") {
		t.Errorf("unknown tag: %v", err)
	}
}

func TestFetch_RefusesOptionLookalikes(t *testing.T) {
	isolate(t)
	cache := filepath.Join(t.TempDir(), "cache")
	for _, p := range []Pin{{"-evil", "v1"}, {"/x", "--upload-pack=touch /tmp/pwned"}, {"--config=x", "-b"}} {
		if _, err := Fetch(p, cache, false); err == nil {
			t.Errorf("Fetch(%v) must fail", p)
		}
	}
	if _, err := os.Stat(cache); err == nil {
		t.Error("git was reached: the cache directory was created")
	}
	if _, err := ListTags("-evil"); err == nil {
		t.Error("ListTags must refuse a hub starting with -")
	}
}

func TestFetch_CacheAndRefresh(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	pin := Pin{h, "v1.0.0"}
	d1, err := Fetch(pin, cache, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(d1, cache) {
		t.Errorf("the hub is cached outside the cache directory: %s", d1)
	}
	if exists(d1, ".git") {
		t.Error(".git kept in the cache")
	}
	files := map[string]string{}
	for k, v := range baseHub {
		files[k] = v
	}
	files["CONVENTIONS.md"] = "# changed\n"
	commitHub(t, h, "v1.0.0", files, nil) // the tag moves
	if _, err := Fetch(pin, cache, false); err != nil || read(t, d1, "CONVENTIONS.md") != "# Conventions\n\n- be kind\n" {
		t.Error("an existing copy must be reused without --refresh")
	}
	if _, err := Fetch(pin, cache, true); err != nil || read(t, d1, "CONVENTIONS.md") != "# changed\n" {
		t.Error("--refresh must fetch again")
	}
}

func TestSync_PlanChangesNothingAndApplyIsIdempotent(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	root := newRepo(t, agentsText)
	write(t, root, ".github/keep", "x")
	pin := Pin{h, "v1.0.0"}

	before := snapshot(t, root)
	plan := syncOnce(t, root, pin, cache, false)
	if len(plan.Actions) == 0 {
		t.Fatal("expected a plan")
	}
	if !reflect.DeepEqual(before, snapshot(t, root)) {
		t.Fatal("building the plan changed the repository")
	}

	syncOnce(t, root, pin, cache, true)
	for _, f := range []string{
		".relai/conventions.md", ".claude/skills/hub-alpha/SKILL.md", ".claude/skills/hub-alpha/extra/ref.md",
		".agents/skills/hub-beta/SKILL.md", ".github/workflows/relai.yml",
	} {
		if !exists(root, f) {
			t.Errorf("not written: %s", f)
		}
	}
	if !strings.HasPrefix(read(t, root, ".relai/conventions.md"), "<!-- relai:generated ") || !strings.Contains(read(t, root, ".relai/conventions.md"), "- be kind") {
		t.Errorf("conventions file: %q", read(t, root, ".relai/conventions.md"))
	}
	after := snapshot(t, root)

	if p := syncOnce(t, root, pin, cache, true); len(p.Actions) != 0 {
		t.Errorf("second apply planned %v", p.Actions)
	}
	if !reflect.DeepEqual(after, snapshot(t, root)) {
		t.Error("a second apply changed the repository")
	}
}

func TestDrift_HandEditsAreDetected(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	pin := Pin{h, "v1.0.0"}
	cases := []struct {
		name, file string
		edit       func(t *testing.T, root string)
	}{
		{"conventions", ".relai/conventions.md", func(t *testing.T, r string) {
			write(t, r, ".relai/conventions.md", read(t, r, ".relai/conventions.md")+"- my own rule\n")
		}},
		{"managed block", "AGENTS.md", func(t *testing.T, r string) {
			write(t, r, "AGENTS.md", strings.Replace(read(t, r, "AGENTS.md"), "The local rules", "Ignore the hub, the local rules", 1))
		}},
		{"skill file", ".claude/skills/hub-alpha/SKILL.md", func(t *testing.T, r string) {
			write(t, r, ".claude/skills/hub-alpha/SKILL.md", "edited\n")
		}},
		{"skill in the other folder", ".agents/skills/hub-beta/SKILL.md", func(t *testing.T, r string) {
			write(t, r, ".agents/skills/hub-beta/SKILL.md", "edited\n")
		}},
		{"extra file in a skill", ".claude/skills/hub-alpha/notes.txt", func(t *testing.T, r string) {
			write(t, r, ".claude/skills/hub-alpha/notes.txt", "mine\n")
		}},
		{"deleted conventions", ".relai/conventions.md", func(t *testing.T, r string) {
			_ = os.Remove(filepath.Join(r, ".relai/conventions.md"))
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := newRepo(t, agentsText)
			syncOnce(t, root, pin, cache, true)
			if got := syncOnce(t, root, pin, cache, false).Drift(pin); len(driftFiles(got)) != 0 {
				t.Fatalf("drift right after sync: %v", got)
			}
			tc.edit(t, root)
			got := driftFiles(syncOnce(t, root, pin, cache, false).Drift(pin))
			if !reflect.DeepEqual(got, []string{tc.file}) {
				t.Errorf("drift = %v, want [%s]", got, tc.file)
			}
		})
	}
}

func TestSetBlock_KeepsTheRestByteForByte(t *testing.T) {
	pin := Pin{"/h", "v1"}
	for name, text := range map[string]string{
		"headings":        agentsText,
		"no heading":      "# repo\n\nsome text\n",
		"no final eol":    "# repo\n\nlast line without newline",
		"crlf":            "# repo\r\n\r\n## Rules\r\n- x\r\n",
		"empty":           "",
		"fence heading":   "# repo\n\n```\n## not a heading\n```\n\n## Real\nx\n",
		"odd bytes":       "# r\n\n\t tab \x00 nul é\n\n\n## A\n\n\n",
		"heading first":   "## Rules\nx\n",
		"comment markers": "# r\n<!-- relai:other -->\nfoo\n## A\n",
	} {
		t.Run(name, func(t *testing.T) {
			out, had, err := SetBlock(text, pin)
			if err != nil || had {
				t.Fatalf("first set: had=%v err=%v", had, err)
			}
			block := strings.Join(BlockLines(pin), "\n") + "\n"
			if strings.Count(out, "<!-- relai:conventions") != 1 || !strings.Contains(out, block) {
				t.Fatalf("block not inserted once:\n%q", out)
			}
			// What is outside the markers, joined back, is the original (ignoring the blank lines added for spacing).
			rest := strings.Replace(out, block, "", 1)
			if strings.ReplaceAll(rest, "\n", "") != strings.ReplaceAll(text, "\n", "") {
				t.Errorf("text outside the block changed:\n got %q\nwant %q", rest, text)
			}
			again, had, err := SetBlock(out, pin)
			if err != nil || !had || again != out {
				t.Errorf("not idempotent: had=%v err=%v\n%q\n%q", had, err, out, again)
			}
			// A new tag rewrites only the block.
			moved, _, err := SetBlock(out, Pin{"/h", "v2"})
			if err != nil || strings.Replace(moved, "/h@v2", "/h@v1", 1) != out {
				t.Errorf("re-pin changed more than the block:\n%q\n%q", out, moved)
			}
		})
	}
	if _, _, err := SetBlock("# r\n<!-- relai:conventions x -->\ntext\n", pin); err == nil {
		t.Error("a start marker without an end marker must be an error")
	}
	out, _, _ := SetBlock("# repo\n\nintro\n\n## Rules\nx\n", pin)
	if i, j := strings.Index(out, "relai:conventions"), strings.Index(out, "## Rules"); i < 0 || j < i {
		t.Errorf("the block must come before the first ## heading:\n%s", out)
	}
}

func TestSync_AgentsMdAbsentIsNotCreated(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	root := newRepo(t, "")
	p := syncOnce(t, root, Pin{h, "v1.0.0"}, cache, true)
	if exists(root, "AGENTS.md") {
		t.Fatal("AGENTS.md was created")
	}
	if !strings.Contains(strings.Join(p.Notes, "\n"), "AGENTS.md not found") {
		t.Errorf("not reported: %v", p.Notes)
	}
	if !exists(root, ".relai/conventions.md") {
		t.Error("the rest must still be vendored")
	}
}

func TestSync_AgentsMdThroughAnInsideSymlinkOnly(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	pin := Pin{h, "v1.0.0"}

	root := newRepo(t, "")
	write(t, root, "docs/real.md", agentsText)
	if err := os.Symlink("docs/real.md", filepath.Join(root, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	syncOnce(t, root, pin, cache, true)
	if st, _ := os.Lstat(filepath.Join(root, "AGENTS.md")); st.Mode()&fs.ModeSymlink == 0 {
		t.Error("the link was replaced")
	}
	if !strings.Contains(read(t, root, "docs/real.md"), "relai:conventions") {
		t.Error("the block must be written through the link")
	}

	outside := t.TempDir()
	write(t, outside, "secret.md", "# outside\n")
	root2 := newRepo(t, "")
	if err := os.Symlink(filepath.Join(outside, "secret.md"), filepath.Join(root2, "AGENTS.md")); err != nil {
		t.Fatal(err)
	}
	p := syncOnce(t, root2, pin, cache, true)
	if read(t, outside, "secret.md") != "# outside\n" {
		t.Fatal("a file outside the repository was written")
	}
	if !strings.Contains(strings.Join(p.Skipped, "\n"), "leaves the repository") {
		t.Errorf("not reported: %v", p.Skipped)
	}
}

func TestSync_SkillRemovedAtANewerTag(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	root := newRepo(t, agentsText)
	write(t, root, ".claude/skills/my-own/SKILL.md", "mine\n")
	syncOnce(t, root, Pin{h, "v1.0.0"}, cache, true)
	if !exists(root, ".claude/skills/hub-beta/SKILL.md") || !exists(root, ".agents/skills/hub-beta/SKILL.md") {
		t.Fatal("beta not installed")
	}

	v2 := map[string]string{}
	for k, v := range baseHub {
		if !strings.HasPrefix(k, "skills/beta") {
			v2[k] = v
		}
	}
	v2["skills/alpha/SKILL.md"] = "alpha v2\n"
	commitHub(t, h, "v2.0.0", v2, nil)

	p := syncOnce(t, root, Pin{h, "v2.0.0"}, cache, false)
	if got := driftFiles(p.Drift(Pin{h, "v2.0.0"})); !contains(got, ".claude/skills/hub-beta/SKILL.md") {
		t.Errorf("the removal is not reported as drift: %v", got)
	}
	syncOnce(t, root, Pin{h, "v2.0.0"}, cache, true)
	for _, d := range []string{".claude/skills/hub-beta", ".agents/skills/hub-beta"} {
		if exists(root, d) {
			t.Errorf("%s was not removed", d)
		}
	}
	if read(t, root, ".claude/skills/hub-alpha/SKILL.md") != "alpha v2\n" {
		t.Error("alpha not updated")
	}
	if read(t, root, ".claude/skills/my-own/SKILL.md") != "mine\n" {
		t.Error("a local skill was touched")
	}
	if !strings.Contains(read(t, root, "AGENTS.md"), "@v2.0.0") {
		t.Error("the block must carry the new tag")
	}
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func TestLoad_SymlinksAreSkippedAndReported(t *testing.T) {
	isolate(t)
	outside := t.TempDir()
	write(t, outside, "passwd", "root:x\n")
	write(t, outside, "dir/inner", "inner\n")
	files := map[string]string{"CONVENTIONS.md": "# c\n", "skills/real/SKILL.md": "ok\n"}
	links := map[string]string{
		"skills/real/leak.md": filepath.Join(outside, "passwd"),
		"skills/real/sub":     filepath.Join(outside, "dir"),
		"skills/linked":       outside,
		"ci/github.yml":       filepath.Join(outside, "passwd"),
	}
	h := newHub(t, "v1.0.0", map[string]string{"x": "x"})
	commitHub(t, h, "v1.0.0", files, links)

	cache := t.TempDir()
	root := newRepo(t, agentsText)
	write(t, root, ".github/keep", "x")
	pin := Pin{h, "v1.0.0"}
	p := syncOnce(t, root, pin, cache, true)

	report := strings.Join(p.Skipped, "\n")
	for _, want := range []string{"skills/real/leak.md", "skills/real/sub", "skills/linked", "ci/github.yml"} {
		if !strings.Contains(report, want) || !strings.Contains(report, "symbolic link") {
			t.Errorf("%s not reported:\n%s", want, report)
		}
	}
	for _, f := range []string{".claude/skills/hub-real/leak.md", ".claude/skills/hub-real/sub", ".claude/skills/hub-real/sub/inner", ".claude/skills/hub-linked", ".github/workflows/relai.yml"} {
		if exists(root, f) {
			t.Errorf("%s was written from a symlink", f)
		}
	}
	if !exists(root, ".claude/skills/hub-real/SKILL.md") {
		t.Error("the regular file must be installed")
	}

	// A CONVENTIONS.md that is a link is refused outright.
	commitHub(t, h, "v2.0.0", map[string]string{}, map[string]string{"CONVENTIONS.md": filepath.Join(outside, "passwd")})
	d, err := Fetch(Pin{h, "v2.0.0"}, cache, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Load(d); err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Errorf("a CONVENTIONS.md symlink must be refused: %v", err)
	}
}

func TestLoad_BigFilesAreSkippedAndReported(t *testing.T) {
	isolate(t)
	files := map[string]string{
		"CONVENTIONS.md":       "# c\n",
		"skills/s/SKILL.md":    "ok\n",
		"skills/s/big.bin":     strings.Repeat("x", MaxFileSize+1),
		"skills/s/exactly.bin": strings.Repeat("x", MaxFileSize),
	}
	h := newHub(t, "v1.0.0", files)
	d, err := Fetch(Pin{h, "v1.0.0"}, t.TempDir(), false)
	if err != nil {
		t.Fatal(err)
	}
	c, err := Load(d)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(c.Skipped, "\n"), "skills/s/big.bin: larger than 1 MiB") {
		t.Errorf("not reported: %v", c.Skipped)
	}
	var names []string
	for _, f := range c.Skills[0].Files {
		names = append(names, f.Rel)
	}
	if !reflect.DeepEqual(names, []string{"SKILL.md", "exactly.bin"}) {
		t.Errorf("files = %v", names)
	}

	files["CONVENTIONS.md"] = strings.Repeat("x", MaxFileSize+1)
	commitHub(t, h, "v2.0.0", files, nil)
	d2, _ := Fetch(Pin{h, "v2.0.0"}, t.TempDir(), false)
	if _, err := Load(d2); err == nil {
		t.Error("an oversized CONVENTIONS.md must be refused")
	}
}

func TestLoad_UnsafeNamesNeverReachTheRepository(t *testing.T) {
	isolate(t)
	files := map[string]string{
		"CONVENTIONS.md":          "# c\n",
		"skills/ok/SKILL.md":      "ok\n",
		"skills/..evil/SKILL.md":  "evil\n",
		"skills/a..b/SKILL.md":    "evil\n",
		"skills/-rf/SKILL.md":     "evil\n",
		"skills/.hidden/SKILL.md": "evil\n",
		"skills/sp ace/SKILL.md":  "evil\n",
		"skills/ok/..x/y.md":      "inside file names with .. are only dots, kept below the skill folder\n",
	}
	h := newHub(t, "v1.0.0", files)
	cache := t.TempDir()
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "relai.yaml", "id: x\n")
	write(t, root, "AGENTS.md", agentsText)
	p := syncOnce(t, root, Pin{h, "v1.0.0"}, cache, true)

	if len(p.Skipped) != 5 {
		t.Errorf("want 5 skipped names, got %v", p.Skipped)
	}
	for path := range snapshot(t, parent) {
		if strings.Contains(path, "evil") || strings.Contains(path, "hub--") || strings.Contains(path, "hub-.") && !strings.Contains(path, "hub-ok") {
			t.Errorf("written from an unsafe name: %s", path)
		}
	}
	for _, e := range []string{"evil", "a..b", "rf", "hidden", "space"} {
		if exists(root, ".claude/skills/hub-"+e) {
			t.Errorf("hub-%s installed", e)
		}
	}
	if !exists(root, ".claude/skills/hub-ok/SKILL.md") {
		t.Error("a safe skill must still be installed")
	}
	if got := snapshot(t, parent); len(got) == 0 {
		t.Fatal("empty snapshot")
	}
}

func TestApply_RefusesPathsOutsideItsPlaceEvenInAHandMadePlan(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo")
	if err := os.MkdirAll(filepath.Join(root, ".claude/skills/hub-x"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, "src/main.go", "package main\n")
	write(t, root, "AGENTS.md", "# a\n")
	abs := filepath.Join(parent, "abs.txt")
	bad := []Action{
		{Op: "write", Kind: KindSkill, Path: "../escape.txt", Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: abs, Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: "/etc/relai-test", Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: ".claude/skills/hub-x/../../../escape.txt", Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: ".claude/skills/hub-../x", Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: ".claude/skills/hub-a..b/x", Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: ".claude/skills/hub-x/..\\..\\escape", Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: ".claude/skills/other/x", Data: []byte("x")},
		{Op: "write", Kind: KindSkill, Path: "src/main.go", Data: []byte("x")},
		{Op: "remove", Kind: KindSkill, Path: "src/main.go"},
		{Op: "remove", Kind: KindSkill, Path: "AGENTS.md"},
		{Op: "remove", Kind: KindSkill, Path: ".claude/skills/hub-x/../../../src/main.go"},
		{Op: "remove", Kind: KindSkill, Path: ".claude/skills/not-hub/f"},
	}
	for _, a := range bad {
		before := snapshot(t, parent)
		// A good action first: a bad plan must change nothing, not even the good part of it.
		good := Action{Op: "write", Kind: KindConventions, Path: ConventionsFile, Data: []byte("ok")}
		done, err := Apply(root, &Plan{Actions: []Action{good, a}})
		if err == nil || len(done) != 0 {
			t.Errorf("%v: err=%v done=%v", a, err, done)
		}
		if !reflect.DeepEqual(before, snapshot(t, parent)) {
			t.Errorf("%v changed the file system", a)
		}
	}
}

func TestApply_NeverWritesThroughASymlinkedFolder(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	outside := t.TempDir()
	root := newRepo(t, agentsText)
	if err := os.MkdirAll(filepath.Join(root, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".claude/skills")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, ".relai")); err != nil {
		t.Fatal(err)
	}
	pin := Pin{h, "v1.0.0"}
	dir, _ := Fetch(pin, cache, false)
	c, _ := Load(dir)
	plan, err := BuildPlan(root, pin, c)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, plan); err == nil {
		t.Error("writing through .relai (a link) must fail")
	}
	entries, _ := os.ReadDir(outside)
	if len(entries) != 0 {
		t.Errorf("something was written outside the repository: %v", entries)
	}
	if !strings.Contains(strings.Join(plan.Skipped, "\n"), ".claude/skills: not a plain folder") {
		t.Errorf("linked skills folder not reported: %v", plan.Skipped)
	}
}

func TestApply_RemovesOnlyRealHubFolders(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", map[string]string{"CONVENTIONS.md": "# c\n"}) // a hub with no skills
	outside := t.TempDir()
	write(t, outside, "precious/data.txt", "keep\n")
	write(t, outside, "file.txt", "keep\n")

	root := newRepo(t, agentsText)
	// hub-linked: a link to a folder outside. hub-real: a real folder with a link inside it.
	if err := os.MkdirAll(filepath.Join(root, ".claude/skills/hub-real"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, root, ".claude/skills/hub-real/SKILL.md", "old\n")
	if err := os.Symlink(filepath.Join(outside, "file.txt"), filepath.Join(root, ".claude/skills/hub-real/link.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "precious"), filepath.Join(root, ".claude/skills/hub-linked")); err != nil {
		t.Fatal(err)
	}
	write(t, root, ".claude/skills/not-hub/keep.md", "mine\n")

	p := syncOnce(t, root, Pin{h, "v1.0.0"}, cache, true)
	if exists(root, ".claude/skills/hub-real") {
		t.Error("the real hub-* folder must be removed")
	}
	if read(t, outside, "precious/data.txt") != "keep\n" || read(t, outside, "file.txt") != "keep\n" {
		t.Fatal("a file outside the repository was removed or changed")
	}
	if st, err := os.Lstat(filepath.Join(root, ".claude/skills/hub-linked")); err != nil || st.Mode()&fs.ModeSymlink == 0 {
		t.Error("the hub-* symlink must be left alone")
	}
	if !strings.Contains(strings.Join(p.Skipped, "\n"), "hub-linked") {
		t.Errorf("not reported: %v", p.Skipped)
	}
	if read(t, root, ".claude/skills/not-hub/keep.md") != "mine\n" {
		t.Error("a non-hub skill was touched")
	}
}

func TestCI_InstalledOnceAndOnlyWhereTheRepoUsesIt(t *testing.T) {
	isolate(t)
	cache := t.TempDir()
	h := newHub(t, "v1.0.0", baseHub)
	pin := Pin{h, "v1.0.0"}

	none := newRepo(t, agentsText)
	syncOnce(t, none, pin, cache, true)
	for _, d := range []string{".github", ".forgejo", ".gitea"} {
		if exists(none, d) {
			t.Errorf("%s created in a repository with no CI", d)
		}
	}

	gh := newRepo(t, agentsText)
	write(t, gh, ".github/dependabot.yml", "x")
	syncOnce(t, gh, pin, cache, true)
	if read(t, gh, ".github/workflows/relai.yml") != "name: relai\n" {
		t.Error("github template missing")
	}
	write(t, gh, ".github/workflows/relai.yml", "my edit\n")
	syncOnce(t, gh, pin, cache, true)
	if read(t, gh, ".github/workflows/relai.yml") != "my edit\n" {
		t.Error("an installed CI file was overwritten")
	}
	if got := driftFiles(syncOnce(t, gh, pin, cache, false).Drift(pin)); len(got) != 0 {
		t.Errorf("an edited CI file is not drift: %v", got)
	}

	fj := newRepo(t, agentsText)
	write(t, fj, ".forgejo/x", "x")
	syncOnce(t, fj, pin, cache, true)
	if read(t, fj, ".forgejo/workflows/relai.yml") != "name: relai-forgejo\n" {
		t.Error("forgejo template missing")
	}
	gt := newRepo(t, agentsText)
	write(t, gt, ".gitea/x", "x")
	syncOnce(t, gt, pin, cache, true)
	if !exists(gt, ".gitea/workflows/relai.yml") {
		t.Error("gitea template missing")
	}

	gl := newRepo(t, agentsText)
	write(t, gl, ".gitlab-ci.yml", "stages: [test]\n")
	p := syncOnce(t, gl, pin, cache, true)
	if read(t, gl, ".gitlab-ci.yml") != "stages: [test]\n" {
		t.Error(".gitlab-ci.yml was edited")
	}
	if !strings.Contains(strings.Join(p.Notes, "\n"), "relai check") {
		t.Errorf("GitLab lines not printed: %v", p.Notes)
	}
}

func TestVersions(t *testing.T) {
	cases := []struct {
		cur  string
		tags []string
		want string
		ok   bool
	}{
		{"v1.9.0", []string{"v1.10.0", "v1.2.0"}, "v1.10.0", true},
		{"v1.10.0", []string{"v1.9.0", "v1.10.0"}, "", false},
		{"v1.0", []string{"v1.0.0"}, "", false},
		{"v1.0.0", []string{"v2", "v10.0.0", "v9.9.9", "nightly", "v1.0.0-rc1"}, "v10.0.0", true},
		{"main", []string{"v1.0.0"}, "", false},
	}
	for _, c := range cases {
		got, ok := Newest(c.cur, c.tags)
		if got != c.want || ok != c.ok {
			t.Errorf("Newest(%s, %v) = %q, %v; want %q, %v", c.cur, c.tags, got, ok, c.want, c.ok)
		}
	}
	a, _ := ParseVersion("v1.10.0")
	b, _ := ParseVersion("v1.9.0")
	if CompareVersions(a, b) <= 0 {
		t.Error("v1.10.0 must be higher than v1.9.0")
	}
}

func TestListTags(t *testing.T) {
	isolate(t)
	h := newHub(t, "v1.0.0", baseHub)
	commitHub(t, h, "v1.10.0", baseHub, nil)
	tags, err := ListTags(h)
	if err != nil {
		t.Fatal(err)
	}
	if n, ok := Newest("v1.9.0", tags); !ok || n != "v1.10.0" {
		t.Errorf("tags %v: newest %q %v", tags, n, ok)
	}
	if _, err := ListTags(filepath.Join(t.TempDir(), "nope")); err == nil || strings.Contains(err.Error(), "fatal") {
		t.Errorf("missing hub: %v", err)
	}
}

func TestOverrides(t *testing.T) {
	got := Overrides("# a\nOverride: skip rule 3 because X\n- Override: other\nnot an Override: here\n")
	if !reflect.DeepEqual(got, []string{"Override: skip rule 3 because X", "Override: other"}) {
		t.Errorf("%v", got)
	}
}

func TestInitHub(t *testing.T) {
	isolate(t)
	dir := filepath.Join(t.TempDir(), "hub")
	created, err := InitHub(dir)
	if err != nil || len(created) < 5 {
		t.Fatalf("%v %v", created, err)
	}
	c, err := Load(dir)
	if err != nil || len(c.Skills) != 1 || len(c.CI) != 3 {
		t.Errorf("a new hub must load: %v %+v", err, c)
	}
	if _, err := InitHub(dir); err == nil || !strings.Contains(err.Error(), "not empty") {
		t.Errorf("a non-empty directory must be refused: %v", err)
	}
	empty := t.TempDir()
	if _, err := InitHub(empty); err != nil {
		t.Errorf("an existing empty directory is fine: %v", err)
	}
	before := snapshot(t, dir)
	_, _ = InitHub(dir)
	if !reflect.DeepEqual(before, snapshot(t, dir)) {
		t.Error("a refused init changed the directory")
	}
}
