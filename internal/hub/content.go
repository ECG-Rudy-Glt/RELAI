package hub

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// MaxFileSize is the largest hub file that is read: bigger ones are skipped and reported.
const MaxFileSize = 1 << 20

// File is one regular file of the hub.
type File struct {
	Rel  string // slash-separated, relative to its skill folder
	Data []byte
	Exec bool
}

// Skill is one folder of the hub's skills/.
type Skill struct {
	Name  string
	Files []File
}

// Content is what a hub offers at one tag.
type Content struct {
	Conventions []byte
	Skills      []Skill
	CI          map[string][]byte // "github", "forgejo", "gitlab"
	Skipped     []string          // what was not read, and why
}

var nameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// SafeName reports whether name can be a skill folder name: letters, digits, dot, underscore and
// dash, starting with a letter or digit, and never containing "..".
func SafeName(name string) bool {
	return nameRe.MatchString(name) && !strings.Contains(name, "..") && len(name) <= 64
}

// safeRel reports whether rel is a clean relative slash path with no "..", no absolute form, no
// backslash and no NUL, as every path taken from the hub must be.
func safeRel(rel string) bool {
	if rel == "" || strings.HasPrefix(rel, "/") || strings.ContainsAny(rel, "\\\x00") || filepath.IsAbs(rel) {
		return false
	}
	for _, part := range strings.Split(rel, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

// readRegular reads a regular file that is not a symbolic link and not bigger than MaxFileSize.
// A reason is returned instead of data when the file must be skipped.
func readRegular(path string) (data []byte, exec bool, skip string, err error) {
	st, err := os.Lstat(path)
	if err != nil {
		return nil, false, "", err
	}
	switch {
	case st.Mode()&fs.ModeSymlink != 0:
		return nil, false, "symbolic link, not followed", nil
	case !st.Mode().IsRegular():
		return nil, false, "not a regular file", nil
	case st.Size() > MaxFileSize:
		return nil, false, fmt.Sprintf("larger than %d MiB", MaxFileSize>>20), nil
	}
	data, err = os.ReadFile(path)
	return data, st.Mode().Perm()&0o111 != 0, "", err
}

// Load reads the hub checked out in dir. CONVENTIONS.md is required; everything else is optional.
// Nothing is followed through a symbolic link and nothing is read twice.
func Load(dir string) (*Content, error) {
	c := &Content{CI: map[string][]byte{}}
	data, _, skip, err := readRegular(filepath.Join(dir, "CONVENTIONS.md"))
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, errors.New("the hub has no CONVENTIONS.md at this tag")
	case err != nil:
		return nil, fmt.Errorf("cannot read CONVENTIONS.md of the hub: %w", err)
	case skip != "":
		return nil, fmt.Errorf("CONVENTIONS.md of the hub cannot be used: %s", skip)
	}
	c.Conventions = data

	for _, kind := range []string{"github", "forgejo", "gitlab"} {
		b, _, skip, err := readRegular(filepath.Join(dir, "ci", kind+".yml"))
		switch {
		case err != nil: // absent
		case skip != "":
			c.Skipped = append(c.Skipped, fmt.Sprintf("ci/%s.yml: %s", kind, skip))
		default:
			c.CI[kind] = b
		}
	}

	root := filepath.Join(dir, "skills")
	if st, err := os.Lstat(root); err != nil || st.Mode()&fs.ModeSymlink != 0 || !st.IsDir() {
		if err == nil {
			c.Skipped = append(c.Skipped, "skills: not a plain folder, no skill installed")
		}
		return c, nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, fmt.Errorf("cannot read skills/ of the hub: %w", err)
	}
	for _, e := range entries { // ReadDir is sorted by name
		name := e.Name()
		switch {
		case e.Type()&fs.ModeSymlink != 0:
			c.Skipped = append(c.Skipped, "skills/"+name+": symbolic link, not followed")
			continue
		case !e.IsDir():
			continue // README.md and the like
		case !SafeName(name):
			c.Skipped = append(c.Skipped, fmt.Sprintf("skills/%s: not a usable skill name (letters, digits, . _ -, never ..)", name))
			continue
		}
		sk := Skill{Name: name}
		base := filepath.Join(root, name)
		werr := filepath.WalkDir(base, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return werr
			}
			rel, rerr := filepath.Rel(base, p)
			if rerr != nil {
				return rerr
			}
			if rel == "." {
				return nil
			}
			rel = filepath.ToSlash(rel)
			shown := "skills/" + name + "/" + rel
			if d.Type()&fs.ModeSymlink != 0 {
				c.Skipped = append(c.Skipped, shown+": symbolic link, not followed")
				return nil // WalkDir does not descend into it
			}
			if d.IsDir() {
				return nil
			}
			if !safeRel(rel) {
				c.Skipped = append(c.Skipped, shown+": unsafe path")
				return nil
			}
			b, ex, skip, err := readRegular(p)
			if err != nil {
				return err
			}
			if skip != "" {
				c.Skipped = append(c.Skipped, shown+": "+skip)
				return nil
			}
			sk.Files = append(sk.Files, File{Rel: rel, Data: b, Exec: ex})
			return nil
		})
		if werr != nil {
			return nil, fmt.Errorf("cannot read skills/%s of the hub: %w", name, werr)
		}
		sort.Slice(sk.Files, func(i, j int) bool { return sk.Files[i].Rel < sk.Files[j].Rel })
		if len(sk.Files) > 0 {
			c.Skills = append(c.Skills, sk)
		}
	}
	return c, nil
}
