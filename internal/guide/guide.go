// Package guide holds the how-to that lets any AI agent use relai with no explanation, and installs
// it as a user-level Agent Skill.
package guide

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

//go:embed guide.md
var text string

// Text returns the guide, the same for every agent.
func Text() string { return text }

// Skill returns the guide as an Agent Skill: YAML frontmatter followed by the text.
func Skill() string {
	return "---\nname: relai\n" +
		"description: Use at the start of a session in a repository, before building something another project may already provide, and when recording a decision or a pitfall: how to use the relai CLI.\n" +
		"---\n\n" + text
}

// Targets are the skill files, relative to the home directory.
var Targets = []string{
	filepath.Join(".claude", "skills", "relai", "SKILL.md"),
	filepath.Join(".agents", "skills", "relai", "SKILL.md"),
}

// Status says what happened to one target.
type Status string

const (
	Written  Status = "written"
	UpToDate Status = "up to date"
	Differs  Status = "exists and differs"
	Refused  Status = "refused"
)

// Result is the outcome for one target.
type Result struct {
	Path   string
	Status Status
	Err    error
}

// Install writes the skill under home. It checks every target first and writes nothing when one
// differs (unless force) or is a symbolic link; a refusal is reported as the returned error.
func Install(home string, force bool) ([]Result, error) {
	want := []byte(Skill())
	var results []Result
	var problems []string
	pending := map[int]bool{}
	for i, rel := range Targets {
		p := filepath.Join(home, rel)
		r := Result{Path: p}
		switch st, err := inspect(home, rel, want); {
		case err != nil:
			r.Status, r.Err = Refused, err
			problems = append(problems, fmt.Sprintf("%s: %v", p, err))
		case st == UpToDate:
			r.Status = UpToDate
		case st == Differs && !force:
			r.Status = Differs
			problems = append(problems, p+" exists and differs (use --force to overwrite)")
		default:
			pending[i] = true
			r.Status = Written
		}
		results = append(results, r)
	}
	if len(problems) > 0 {
		for i := range results {
			if results[i].Status == Written {
				results[i].Status = "not written"
			}
		}
		return results, fmt.Errorf("nothing written: %s", strings.Join(problems, "; "))
	}
	for i, rel := range Targets {
		if !pending[i] {
			continue
		}
		if err := write(filepath.Join(home, rel), want); err != nil {
			results[i].Status, results[i].Err = Refused, err
			return results, err
		}
	}
	return results, nil
}

// inspect compares the target with want; it returns an error for a symbolic link anywhere from the
// relai directory down to the file.
func inspect(home, rel string, want []byte) (Status, error) {
	p := filepath.Join(home, rel)
	for _, q := range []string{filepath.Dir(p), p} {
		fi, err := os.Lstat(q)
		if os.IsNotExist(err) {
			return Written, nil
		}
		if err != nil {
			return "", err
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("%s is a symbolic link: refusing to follow it", q)
		}
	}
	have, err := os.ReadFile(p)
	if err != nil {
		return "", err
	}
	if string(have) == string(want) {
		return UpToDate, nil
	}
	return Differs, nil
}

func write(p string, data []byte) error {
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".SKILL-*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), p)
}
