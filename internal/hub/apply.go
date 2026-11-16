package hub

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	skillPathRe = regexp.MustCompile(`^\.(claude|agents)/skills/(hub-[A-Za-z0-9][A-Za-z0-9._-]*)/(.+)$`)
	writeExact  = map[string]bool{
		ConventionsFile:                true,
		"AGENTS.md":                    true,
		".github/workflows/relai.yml":  true,
		".forgejo/workflows/relai.yml": true,
		".gitea/workflows/relai.yml":   true,
	}
)

// checkPath is the static rule every action must pass, whatever built the plan: a clean relative
// path, and one of the few places relai sync is allowed to touch. A removal is only ever inside
// a hub-* skill folder.
func checkPath(a Action) error {
	if !safeRel(a.Path) {
		return fmt.Errorf("refusing to touch %q: not a clean path inside the repository", a.Path)
	}
	m := skillPathRe.FindStringSubmatch(a.Path)
	if m != nil && !strings.Contains(m[2], "..") {
		return nil
	}
	if a.Op == "write" && writeExact[a.Path] {
		return nil
	}
	return fmt.Errorf("refusing to %s %q: relai sync only manages its own files", a.Op, a.Path)
}

// noLinkParents fails when a directory on the way to rel (rel itself excluded) is a symbolic link:
// writing through it could leave the repository.
func noLinkParents(root, rel string) error {
	parts := strings.Split(rel, "/")
	cur := root
	for _, part := range parts[:len(parts)-1] {
		cur = filepath.Join(cur, part)
		st, err := os.Lstat(cur)
		if err != nil {
			return nil // does not exist yet: it will be created as a plain folder
		}
		if st.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write %q: %s is a symbolic link", rel, strings.TrimPrefix(cur, root+string(filepath.Separator)))
		}
		if !st.IsDir() {
			return fmt.Errorf("refusing to write %q: %s is not a folder", rel, strings.TrimPrefix(cur, root+string(filepath.Separator)))
		}
	}
	return nil
}

// Apply performs the plan inside root and returns what it did. Every path is checked first, so a
// plan that asks for something outside the allowed places changes nothing. It never commits.
func Apply(root string, p *Plan) ([]Action, error) {
	for _, a := range p.Actions {
		if err := checkPath(a); err != nil {
			return nil, err
		}
	}
	var done []Action
	for _, a := range p.Actions {
		if err := do(root, a); err != nil {
			return done, err
		}
		done = append(done, a)
	}
	return done, nil
}

func do(root string, a Action) error {
	full := filepath.Join(root, filepath.FromSlash(a.Path))
	if err := noLinkParents(root, a.Path); err != nil {
		return err
	}
	if a.Op == "remove" {
		return doRemove(root, a, full)
	}
	if a.Op != "write" {
		return fmt.Errorf("unknown action %q", a.Op)
	}
	st, err := os.Lstat(full)
	exists := err == nil
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	perm := fs.FileMode(0o644)
	if a.Exec {
		perm = 0o755
	}
	switch {
	case a.Kind == KindCI: // once: never replace what is there
		f, err := os.OpenFile(full, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if _, err := f.Write(a.Data); err != nil {
			f.Close()
			return err
		}
		return f.Close()
	case a.Path == "AGENTS.md":
		if exists && st.Mode()&fs.ModeSymlink != 0 && !within(root, full) {
			return errors.New("refusing to write AGENTS.md: it is a symbolic link that leaves the repository")
		}
		if !exists {
			return errors.New("AGENTS.md does not exist: relai sync never creates it")
		}
		return os.WriteFile(full, a.Data, 0o644) // through an inside link: the target is the file
	default:
		if exists && st.Mode()&fs.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write %q: it is a symbolic link", a.Path)
		}
		if exists && !st.Mode().IsRegular() {
			return fmt.Errorf("refusing to write %q: not a regular file", a.Path)
		}
		return os.WriteFile(full, a.Data, perm)
	}
}

func doRemove(root string, a Action, full string) error {
	m := skillPathRe.FindStringSubmatch(a.Path)
	if m == nil {
		return fmt.Errorf("refusing to remove %q", a.Path)
	}
	top := filepath.Join(root, "."+m[1], "skills", m[2])
	if st, err := os.Lstat(top); err != nil || st.Mode()&fs.ModeSymlink != 0 || !st.IsDir() {
		return fmt.Errorf("refusing to remove %q: %s is not a real folder", a.Path, m[2])
	}
	st, err := os.Lstat(full)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if st.IsDir() {
		return fmt.Errorf("refusing to remove %q: it is a folder", a.Path)
	}
	if err := os.Remove(full); err != nil { // a symbolic link is removed itself, never followed
		return err
	}
	for dir := filepath.Dir(full); strings.HasPrefix(dir, top); dir = filepath.Dir(dir) {
		if os.Remove(dir) != nil { // not empty
			break
		}
	}
	return nil
}
