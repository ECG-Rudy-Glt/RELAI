// Package hub vendors shared conventions from a hub repository into a repository.
//
// Everything here is a pure function that returns a plan (what would be written or removed) or a
// value: nothing prints and nothing exits. The only side effects are Fetch (a git clone into the
// user cache directory), Apply (writes the plan inside the repository) and InitHub.
//
// The content of a hub is UNTRUSTED, as the content of a `relai new` template is: names are
// checked, symbolic links are never followed, big files are skipped, and Apply writes only to a
// short list of places inside the repository.
package hub

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// Pin is the hub a repository follows: `conventions: <hub>@<tag>` in relai.yaml.
type Pin struct {
	Hub string // clone URL or local path
	Tag string // a git tag of the hub
}

func (p Pin) String() string { return p.Hub + "@" + p.Tag }

// ParsePin splits value on its last "@" (a clone URL may contain one: git@host:team/hub.git@v1).
// A hub or tag that starts with "-" is refused here, before git is ever called, because git would
// read it as an option.
func ParsePin(value string) (Pin, error) {
	v := strings.TrimSpace(value)
	i := strings.LastIndex(v, "@")
	if v == "" || i < 0 {
		return Pin{}, errors.New("conventions must be written <hub>@<tag>, for example conventions: https://example.org/team/hub.git@v1.0.0")
	}
	p := Pin{Hub: v[:i], Tag: v[i+1:]}
	switch {
	case p.Hub == "" || p.Tag == "":
		return Pin{}, errors.New("conventions must be written <hub>@<tag>: the hub or the tag is empty")
	case strings.HasPrefix(p.Hub, "-") || strings.HasPrefix(p.Tag, "-"):
		return Pin{}, errors.New("conventions: the hub and the tag must not start with \"-\"")
	case strings.ContainsFunc(v, func(r rune) bool { return r < 0x20 || r == 0x7f }), strings.Contains(v, "-->"):
		return Pin{}, errors.New("conventions: the hub and the tag must not contain control characters or \"-->\"")
	}
	return p, nil
}

// gitTimeout bounds every git call, as the rest of relai does.
const gitTimeout = 60 * time.Second

// git runs git with prompts and the "ext::" transport disabled, in the C locale so that the
// messages classified below do not depend on the user's language.
func git(args ...string) (stdout, stderr string, err error) {
	args = append([]string{"--no-optional-locks", "-c", "protocol.ext.allow=never", "-c", "core.quotepath=off"}, args...)
	cmd := exec.Command("git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C")
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	if err := cmd.Start(); err != nil {
		return "", "", err
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return out.String(), errb.String(), err
	case <-time.After(gitTimeout):
		_ = cmd.Process.Kill()
		<-done
		return "", "", errors.New("git timed out")
	}
}

func hubError(p Pin, stderr string) error {
	if strings.Contains(stderr, "Remote branch") && strings.Contains(stderr, "not found") {
		return fmt.Errorf("the hub %s has no tag %q: check the conventions line of relai.yaml", p.Hub, p.Tag)
	}
	return fmt.Errorf("cannot read the hub %s: is it a git repository you can reach?", p.Hub)
}

// CachePath is where the hub at the pinned tag is kept under cacheDir.
func CachePath(cacheDir string, p Pin) string {
	sum := sha256.Sum256([]byte(p.Hub + "\x00" + p.Tag))
	return filepath.Join(cacheDir, "hubs", hex.EncodeToString(sum[:10]))
}

// Fetch makes sure the hub at the pinned tag is in cacheDir and returns its directory (without
// .git). An existing copy is reused unless refresh is true. cacheDir is the user cache directory,
// never the repository.
func Fetch(p Pin, cacheDir string, refresh bool) (string, error) {
	if _, err := ParsePin(p.String()); err != nil { // never reach git with a value ParsePin refuses
		return "", err
	}
	dest := CachePath(cacheDir, p)
	if st, err := os.Stat(dest); err == nil && st.IsDir() && !refresh {
		return dest, nil
	}
	parent := filepath.Dir(dest)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return "", fmt.Errorf("cannot create the cache directory %s: %w", parent, err)
	}
	tmp, err := os.MkdirTemp(parent, ".fetch-")
	if err != nil {
		return "", fmt.Errorf("cannot write to the cache directory %s: %w", parent, err)
	}
	defer os.RemoveAll(tmp)
	clone := filepath.Join(tmp, "hub")
	_, stderr, err := git("clone", "--quiet", "--depth", "1", "--branch", p.Tag, "--", p.Hub, clone)
	if err != nil {
		return "", hubError(p, stderr)
	}
	_ = os.RemoveAll(filepath.Join(clone, ".git"))
	_ = os.RemoveAll(dest)
	if err := os.Rename(clone, dest); err != nil {
		return "", fmt.Errorf("cannot store the hub in the cache: %w", err)
	}
	return dest, nil
}

// ListTags returns the tags of the hub (git ls-remote), in no particular order.
func ListTags(hubSrc string) ([]string, error) {
	if hubSrc == "" || strings.HasPrefix(hubSrc, "-") {
		return nil, errors.New("conventions: the hub must not be empty or start with \"-\"")
	}
	out, stderr, err := git("ls-remote", "--tags", "--refs", "--", hubSrc)
	if err != nil {
		return nil, hubError(Pin{Hub: hubSrc}, stderr)
	}
	var tags []string
	for _, l := range strings.Split(out, "\n") {
		if _, ref, ok := strings.Cut(strings.TrimSpace(l), "\t"); ok && strings.HasPrefix(ref, "refs/tags/") {
			tags = append(tags, strings.TrimPrefix(ref, "refs/tags/"))
		}
	}
	return tags, nil
}

// ParseVersion reads a tag such as v1.10.0 or 2.3 as numbers. Anything else (a branch name, a
// pre-release suffix) is not a version and is never reported as newer.
func ParseVersion(tag string) ([]int, bool) {
	t := strings.TrimPrefix(strings.TrimPrefix(tag, "v"), "V")
	if t == "" {
		return nil, false
	}
	var v []int
	for _, part := range strings.Split(t, ".") {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return nil, false
		}
		n, err := strconv.Atoi(part)
		if err != nil {
			return nil, false
		}
		v = append(v, n)
	}
	return v, true
}

// CompareVersions orders two versions numerically: 1.10.0 is higher than 1.9.0, and 1.2 equals 1.2.0.
func CompareVersions(a, b []int) int {
	for i := 0; i < len(a) || i < len(b); i++ {
		var x, y int
		if i < len(a) {
			x = a[i]
		}
		if i < len(b) {
			y = b[i]
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

// Newest returns the highest version tag that is higher than current, when there is one.
func Newest(current string, tags []string) (string, bool) {
	cur, ok := ParseVersion(current)
	if !ok {
		return "", false
	}
	best, bestV, found := "", cur, false
	for _, t := range tags {
		if v, ok := ParseVersion(t); ok && CompareVersions(v, bestV) > 0 {
			best, bestV, found = t, v, true
		}
	}
	return best, found
}
