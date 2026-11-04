package scan

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Link is what repository From says about repository To. Strong evidence is hard to produce by
// accident (To's remote slug, a relative path to it); weak evidence is the name as a word.
type Link struct {
	From     string   `json:"from"`
	To       string   `json:"to"`
	Strong   int      `json:"strong_files"` // files of From holding strong evidence
	Weak     int      `json:"weak_files"`   // files of From where To's name appears as a word
	Kinds    []string `json:"kinds,omitempty"`
	Evidence []string `json:"evidence,omitempty"` // up to three paths of From, never their content
}

// minNameLength keeps very short names ("ring", "app") from matching everywhere: shorter ones are
// only searched through strong evidence.
const minNameLength = 4

// generated or vendored files say nothing about what a person works with.
var excluded = []string{":!graphify-out", ":!node_modules", ":!vendor", ":!*.lock", ":!package-lock.json", ":!go.sum", ":!*.min.js", ":!*.svg"}

// Links returns the mentions between the repositories, strongest first. A repository is never linked
// to itself.
func Links(repos []Repo) []Link {
	var out []Link
	for _, a := range repos {
		for _, b := range repos {
			if a.Path == b.Path {
				continue
			}
			l := Link{From: a.Name, To: b.Name}
			strong := map[string]bool{}
			if b.Slug != "" {
				for _, f := range grep(a.Path, b.Slug, false) {
					strong[f] = true
					l.addKind("remote URL")
				}
			}
			for _, f := range grepRegexp(a.Path, `\.\./`+regexp.QuoteMeta(b.Name)+`(/|"|'|\s|$)`) {
				strong[f] = true
				l.addKind("relative path")
			}
			l.Strong = len(strong)
			weak := map[string]bool{}
			if len(b.Name) >= minNameLength {
				for _, f := range grep(a.Path, b.Name, true) {
					weak[f] = true
				}
			}
			l.Weak = len(weak)
			if l.Strong == 0 && l.Weak == 0 {
				continue
			}
			var ev []string
			for f := range strong {
				ev = append(ev, f)
			}
			if len(ev) == 0 {
				for f := range weak {
					ev = append(ev, f)
				}
			}
			sort.Strings(ev)
			if len(ev) > 3 {
				ev = ev[:3]
			}
			l.Evidence = ev
			out = append(out, l)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Strong != out[j].Strong {
			return out[i].Strong > out[j].Strong
		}
		if out[i].Weak != out[j].Weak {
			return out[i].Weak > out[j].Weak
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func (l *Link) addKind(k string) {
	for _, x := range l.Kinds {
		if x == k {
			return
		}
	}
	l.Kinds = append(l.Kinds, k)
}

func grep(dir, pattern string, word bool) []string {
	args := []string{"-C", dir, "--no-optional-locks", "grep", "-l", "-I", "-i", "-F"}
	if word {
		args = append(args, "-w")
	}
	return runGrep(args, pattern)
}

func grepRegexp(dir, pattern string) []string {
	return runGrep([]string{"-C", dir, "--no-optional-locks", "grep", "-l", "-I", "-E"}, pattern)
}

func runGrep(args []string, pattern string) []string {
	args = append(args, "-e", pattern, "--", ".")
	for _, e := range excluded {
		args = append(args, e)
	}
	cmd := exec.Command("git", args...)
	done := make(chan []byte, 1)
	go func() { out, _ := cmd.Output(); done <- out }() // exit 1 means no match
	select {
	case out := <-done:
		var files []string
		for _, l := range strings.Split(string(out), "\n") {
			if l = strings.TrimSpace(l); l != "" {
				files = append(files, l)
			}
		}
		return files
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		return nil
	}
}

// Text renders the links for a person. It says in the first lines what a mention is and is not.
func Text(dir string, repos []Repo, links []Link) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Links between %d repositories in %s\n\n", len(repos), dir)
	b.WriteString("These are MENTIONS found in tracked files, not verified dependencies. \"strong\" = the other repository's\nremote URL or a relative path to it; \"weak\" = its name as a word (can be a common word). Confirm before you declare a link.\n\n")
	if len(links) == 0 {
		b.WriteString("(no mention of one repository in another)\n")
		return b.String()
	}
	for _, l := range links {
		ev := strings.Join(l.Kinds, ", ")
		if ev != "" {
			ev = " (" + ev + ")"
		}
		fmt.Fprintf(&b, "- %s -> %s: strong %d file(s)%s, weak %d file(s)\n", l.From, l.To, l.Strong, ev, l.Weak)
		if len(l.Evidence) > 0 {
			fmt.Fprintf(&b, "    e.g. %s\n", strings.Join(l.Evidence, ", "))
		}
	}
	return b.String()
}

// weakThreshold is how many files must mention a name before it is worth proposing without strong evidence.
const weakThreshold = 5

// Propose renders, for each repository, a `links:` block a person can paste into its relai.yaml after
// checking it. Nothing is written anywhere.
func Propose(repos []Repo, links []Link) string {
	by := map[string][]Link{}
	for _, l := range links {
		if l.Strong > 0 || l.Weak >= weakThreshold {
			by[l.From] = append(by[l.From], l)
		}
	}
	var b strings.Builder
	b.WriteString("# Proposed `links:` for each relai.yaml. Check each line, then paste it yourself: nothing was written.\n")
	for _, r := range repos {
		ls := by[r.Name]
		if len(ls) == 0 {
			continue
		}
		fmt.Fprintf(&b, "\n# %s/relai.yaml\nlinks:\n", r.Name)
		for _, l := range ls {
			conf := fmt.Sprintf("weak: %d files", l.Weak)
			if l.Strong > 0 {
				conf = fmt.Sprintf("strong: %d files", l.Strong)
			}
			id := l.To
			for _, o := range repos {
				if o.Name == l.To && o.ID != "" {
					id = o.ID
				}
			}
			fmt.Fprintf(&b, "  - %s   # %s\n", id, conf)
		}
	}
	return b.String()
}

// Prompt is what an agent should do with the mentions: check each one against the files, keep the real
// links, and write them down with the engineer's agreement. The tool found the candidates; it does not decide.
func Prompt(dir string, repos []Repo, links []Link) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You are helping the engineer declare how the repositories in %s relate to each other.\n\n", dir)
	b.WriteString(`Relai found MENTIONS between repositories (below). A mention is not a verified link: a name can be an ordinary
word, a repository can be named in a doc without depending on it. Your job is to separate the real links from the noise.

For each candidate, in order of strength:
1. Open the evidence files listed (only those) in the repository that mentions the other, and read the lines around the mention.
2. Decide: does the first repository really deploy, build, call, include or document the second? Answer yes, no or unsure, with one line of reason.
3. For the "yes" ones, prepare a links: entry (the other repository's relai.yaml id) for the first repository's relai.yaml.
   If a repository has no relai.yaml, say so: it needs "relai init" first. Do not create files before the engineer agrees.
4. Show the engineer one table: candidate, your verdict, the reason. Ask for confirmation, and only then write the relai.yaml changes.
5. Also note capabilities you can see (what a repository provides to the others, what it consumes) and propose provides:/consumes: the same way.

Never copy secrets, tokens or file contents into your answer: name files and say what they show.

`)
	b.WriteString(Text(dir, repos, links))
	b.WriteString("\nRepositories found:\n")
	for _, r := range repos {
		state := "no relai.yaml (run relai init)"
		if r.Relai {
			state = "relai.yaml, id " + r.ID
		}
		fmt.Fprintf(&b, "- %s — %s; last commit %s\n", r.Name, state, r.LastCommit)
	}
	return b.String()
}
