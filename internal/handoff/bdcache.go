package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// Every `bd` run rewrites about 3 MiB of its embedded database, even for a read, and the handoff
// needs three of them. A session-start hook would pay that on every session. So the three answers
// are kept in the user's cache directory, keyed by a fingerprint of the repo's .beads/last-touched
// (which bd changes when it writes and never when it reads), and reused while it is unchanged. The
// cache lives outside the repository: the handoff must not modify it.

const bdCacheMaxAge = time.Hour // belt and braces if some write ever skips last-touched

type bdCache struct {
	FP      string    `json:"fp"`
	At      time.Time `json:"at"`
	Ready   string    `json:"ready"`
	Doing   string    `json:"doing"`
	Blocked string    `json:"blocked"`
}

// bdOutputs returns the raw JSON of `bd ready`, `bd list --status in_progress` and `bd blocked`.
// An error is returned only when `bd ready` itself fails.
func bdOutputs(dir string) (ready, doing, blocked string, err error) {
	fp := bdFingerprint(dir)
	path := bdCachePath(dir)
	if fp != "" && path != "" {
		if b, rerr := os.ReadFile(path); rerr == nil {
			var c bdCache
			if json.Unmarshal(b, &c) == nil && c.FP == fp && time.Since(c.At) < bdCacheMaxAge {
				return c.Ready, c.Doing, c.Blocked, nil
			}
		}
	}
	if ready, err = run(dir, "bd", "ready", "--json"); err != nil {
		return "", "", "", err
	}
	doing = must(run(dir, "bd", "list", "--status", "in_progress", "--json"))
	blocked = must(run(dir, "bd", "blocked", "--json"))
	if fp != "" && path != "" && json.Valid([]byte(ready)) {
		if b, merr := json.Marshal(bdCache{FP: fp, At: time.Now(), Ready: ready, Doing: doing, Blocked: blocked}); merr == nil {
			if os.MkdirAll(filepath.Dir(path), 0o700) == nil {
				_ = os.WriteFile(path, b, 0o600)
			}
		}
	}
	return ready, doing, blocked, nil
}

// bdFingerprint is "" when it cannot be computed (no last-touched file): then nothing is cached.
func bdFingerprint(dir string) string {
	p := filepath.Join(dir, ".beads", "last-touched")
	st, err := os.Stat(p)
	if err != nil {
		return ""
	}
	content, _ := os.ReadFile(p)
	return st.ModTime().UTC().Format(time.RFC3339Nano) + "|" + string(content)
}

func bdCachePath(dir string) string {
	base, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	sum := sha256.Sum256([]byte(dir))
	return filepath.Join(base, "relai", "bd-"+hex.EncodeToString(sum[:8])+".json")
}
