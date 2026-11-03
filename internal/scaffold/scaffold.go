// Package scaffold creates a new repository from a template — a local directory or a clone URL —
// substituting a small set of placeholder tokens, then filling in anything the template didn't
// already provide with the Relai repo contract (internal/initrepo).
//
// Relai ships no template of its own: every call names one explicitly.
package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

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
		return initrepo.Result{}, err
	}
	return initrepo.Init(dir, true)
}

// Copy copies every file from templateDir into dir, preserving relative paths and permissions,
// skipping a .git directory (present when templateDir came from a cloned URL). dir must not
// already exist, or must exist and be empty — a non-empty existing dir is refused, never merged
// into.
func Copy(templateDir, dir string) error {
	if entries, err := os.ReadDir(dir); err == nil {
		if len(entries) > 0 {
			return fmt.Errorf("%s already exists and is not empty", dir)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return filepath.WalkDir(templateDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
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
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, info.Mode().Perm())
	})
}

// Substitute rewrites every regular file under dir in place, parsing it as a text/template and
// executing it against tokens. A file that fails to parse or execute as a template — including
// any binary file, which essentially never parses as one — is left exactly as Copy wrote it.
func Substitute(dir string, tokens Tokens) error {
	return filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		tmpl, err := template.New(filepath.Base(p)).Option("missingkey=error").Parse(string(b))
		if err != nil {
			return nil
		}
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, tokens); err != nil {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return os.WriteFile(p, buf.Bytes(), info.Mode().Perm())
	})
}
