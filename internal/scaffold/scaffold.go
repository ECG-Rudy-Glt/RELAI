// Package scaffold creates a new repository from a template — a local directory or a clone URL —
// substituting a small set of placeholder tokens, then filling in anything the template didn't
// already provide with the Relai repo contract (internal/initrepo).
//
// Relai ships no template of its own: every call names one explicitly.
package scaffold

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
	"github.com/ECG-Rudy-Glt/RELAI/internal/initrepo"
)

// Tokens are the placeholder values substituted into a template's text files.
type Tokens struct {
	RepoName string
}

// New scaffolds dir from templateSrc (a local path or clone URL): fetches the template, copies
// it into dir (which must not already exist and be non-empty), substitutes tokens into its text
// files, then runs the same contract-writing logic as relai init for anything the template did
// not already provide.
func New(templateSrc, dir string, tokens Tokens) (initrepo.Result, error) {
	path, cleanup, err := index.Fetch(templateSrc)
	if err != nil {
		return initrepo.Result{}, fmt.Errorf("template source %s: %w", templateSrc, err)
	}
	defer cleanup()

	if st, err := os.Stat(path); err != nil || !st.IsDir() {
		return initrepo.Result{}, fmt.Errorf("template source not found or not a directory: %s", templateSrc)
	}

	if err := Copy(path, dir); err != nil {
		return initrepo.Result{}, err
	}
	if err := Substitute(dir, tokens); err != nil {
		removeCreated(dir)
		return initrepo.Result{}, err
	}
	r, err := initrepo.Init(dir, true)
	if err != nil {
		removeCreated(dir)
		return r, err
	}
	return r, nil
}

// removeCreated empties dir after a failed run and removes it, so a retry is not refused because
// of a half-filled destination. Copy only ever writes into a directory that was absent or empty,
// so everything in it was put there by this run.
func removeCreated(dir string) { _ = os.RemoveAll(dir) }

// Copy copies every file from templateDir into dir, preserving relative paths and permissions,
// skipping a .git directory (present when templateDir came from a cloned URL). dir must not
// already exist, or must exist and be empty — a non-empty existing dir is refused, never merged
// into. Symbolic links are never read through: a link that stays inside the template is recreated
// as a link, one that points outside it (absolute, or climbing out with ..) makes the copy fail,
// so a template cannot pull a file of the machine that instantiates it into the new repo. dir may
// not be inside templateDir. If the copy fails, what it created is removed.
func Copy(templateDir, dir string) (err error) {
	if entries, rerr := os.ReadDir(dir); rerr == nil {
		if len(entries) > 0 {
			return fmt.Errorf("%s already exists and is not empty", dir)
		}
	} else if !os.IsNotExist(rerr) {
		return rerr
	}
	if inside(templateDir, dir) {
		return fmt.Errorf("destination %s is inside the template %s: copying would recurse", dir, templateDir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			removeCreated(dir)
		}
	}()
	return filepath.WalkDir(templateDir, func(p string, d fs.DirEntry, werr error) error {
		if werr != nil {
			return werr
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(templateDir, p)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dir, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			if filepath.IsAbs(link) || escapes(filepath.Dir(rel), link) {
				return fmt.Errorf("%s is a symbolic link to %s, outside the template: refusing to copy it", rel, link)
			}
			return os.Symlink(link, target)
		case !d.Type().IsRegular():
			return fmt.Errorf("%s is not a regular file, directory or link: refusing to copy it", rel)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(p, target, info.Mode().Perm())
	})
}

// copyFile streams src to dst. The file is created writable and given its final mode afterwards,
// so a read-only template file (0444) can still be substituted later.
func copyFile(src, dst string, perm fs.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm|0o200)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	return os.Chmod(dst, perm)
}

// escapes reports whether the link target, resolved from the directory dirRel, leaves the tree.
func escapes(dirRel, link string) bool {
	r := filepath.Clean(filepath.Join(dirRel, link))
	return r == ".." || strings.HasPrefix(r, ".."+string(filepath.Separator))
}

// inside reports whether path is dir itself or below it, comparing real paths where they exist.
func inside(dir, path string) bool {
	d, p := realAbs(dir), realAbs(path)
	r, err := filepath.Rel(d, p)
	return err == nil && (r == "." || (r != ".." && !strings.HasPrefix(r, ".."+string(filepath.Separator))))
}

// realAbs makes p absolute and resolves symbolic links in its longest existing prefix.
func realAbs(p string) string {
	a, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	rest := ""
	for cur := a; ; cur = filepath.Dir(cur) {
		if r, err := filepath.EvalSymlinks(cur); err == nil {
			return filepath.Join(r, rest)
		}
		if filepath.Dir(cur) == cur {
			return a
		}
		rest = filepath.Join(filepath.Base(cur), rest)
	}
}

// maxSubstitute is the largest file Substitute will read: bigger ones are copied untouched.
const maxSubstitute = 1 << 20

var repoNameToken = regexp.MustCompile(`\{\{\s*\.RepoName\s*\}\}`)

// Substitute rewrites {{.RepoName}} (spaces inside the braces allowed) in every text file under
// dir. Only that token is replaced: any other {{...}} — a GitHub Actions ${{ github.sha }}, a Go
// or Helm template — is left alone, and so is the whole file when it is binary (a NUL byte or
// invalid UTF-8), larger than 1 MiB, or a symbolic link. A file is only written when it changes,
// and keeps its mode, including read-only.
func Substitute(dir string, tokens Tokens) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !d.Type().IsRegular() {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Size() > maxSubstitute {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) || !repoNameToken.Match(b) {
			return nil
		}
		out := repoNameToken.ReplaceAll(b, []byte(tokens.RepoName))
		perm := info.Mode().Perm()
		if perm&0o200 == 0 {
			if err := os.Chmod(p, perm|0o200); err != nil {
				return err
			}
			defer os.Chmod(p, perm)
		}
		return os.WriteFile(p, out, perm|0o200)
	})
}
