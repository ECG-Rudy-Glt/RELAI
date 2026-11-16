package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/handoff"
	"github.com/ECG-Rudy-Glt/RELAI/internal/hub"
	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
	"github.com/ECG-Rudy-Glt/RELAI/internal/validate"
)

// hubCacheDir is where fetched hubs are kept: RELAI_CACHE_DIR when set (tests, CI), else the
// user cache directory. Never the repository.
func hubCacheDir() (string, error) {
	if d := os.Getenv("RELAI_CACHE_DIR"); d != "" {
		return d, nil
	}
	d, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("no user cache directory to keep the hub in: %w", err)
	}
	return filepath.Join(d, "relai"), nil
}

func newFlags(name string, stderr io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(stderr)
	return fs
}

// parseFlags returns an exit code when the command must stop (2 on a bad flag, 0 for -h).
func parseFlags(fs *flag.FlagSet, args []string) (int, bool) {
	if err := fs.Parse(reorderArgs(fs, args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0, true
		}
		return 2, true
	}
	return 0, false
}

// readPin reads `conventions:` of the relai.yaml in dir. ok is false when the repository pins no hub.
func readPin(dir string) (pin hub.Pin, ok bool, err error) {
	b, rerr := os.ReadFile(filepath.Join(dir, "relai.yaml"))
	if rerr != nil {
		return hub.Pin{}, false, fmt.Errorf("no relai.yaml in %s (relai init writes one)", dir)
	}
	v := strings.TrimSpace(index.ParseManifest(b).Conventions)
	if v == "" {
		return hub.Pin{}, false, nil
	}
	pin, err = hub.ParsePin(v)
	return pin, err == nil, err
}

// cmdSync implements `relai sync [--apply] [--refresh] [dir]`.
func cmdSync(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("sync", stderr)
	apply := fs.Bool("apply", false, "do it (default: only show what would change)")
	refresh := fs.Bool("refresh", false, "fetch the hub again instead of using the cached copy")
	if code, stop := parseFlags(fs, args); stop {
		return code
	}
	dir := dirArg(fs)
	fail := func(err error) int { fmt.Fprintln(stderr, "relai:", err); return 1 }

	pin, ok, err := readPin(dir)
	if err != nil {
		return fail(err)
	}
	if !ok {
		return fail(errors.New("relai.yaml has no `conventions:` line: add `conventions: <hub>@<tag>` first"))
	}
	cache, err := hubCacheDir()
	if err != nil {
		return fail(err)
	}
	hubDir, err := hub.Fetch(pin, cache, *refresh)
	if err != nil {
		return fail(err)
	}
	content, err := hub.Load(hubDir)
	if err != nil {
		return fail(err)
	}
	plan, err := hub.BuildPlan(dir, pin, content)
	if err != nil {
		return fail(err)
	}
	if !*apply {
		fmt.Fprint(stdout, plan.Text(false))
		if len(plan.Actions) > 0 {
			fmt.Fprintln(stdout, "Dry run: nothing was written. Run `relai sync --apply` to do it.")
		}
		return 0
	}
	done, err := hub.Apply(dir, plan)
	applied := &hub.Plan{Actions: done, Notes: plan.Notes, Skipped: plan.Skipped}
	fmt.Fprint(stdout, applied.Text(true))
	if err != nil {
		return fail(err)
	}
	if len(done) > 0 {
		fmt.Fprintln(stdout, "Review the changes and commit them: relai never commits.")
	}
	return 0
}

// cmdCheck implements `relai check [--outdated] [--strict] [--json] [dir]`.
func cmdCheck(args []string, stdout, stderr io.Writer) int {
	fs := newFlags("check", stderr)
	outdated := fs.Bool("outdated", false, "warn when the hub has a higher version tag than the pinned one")
	strict := fs.Bool("strict", false, "fail on warnings too")
	asJSON := fs.Bool("json", false, "print the findings as JSON")
	if code, stop := parseFlags(fs, args); stop {
		return code
	}
	dir := dirArg(fs)
	findings := checkFindings(dir, *outdated)
	if *asJSON {
		if findings == nil {
			findings = []validate.Finding{}
		}
		b, _ := json.MarshalIndent(findings, "", "  ")
		fmt.Fprintln(stdout, string(b))
	} else {
		fmt.Fprint(stdout, validate.Text(findings))
	}
	if validate.Failed(findings, *strict) {
		return 1
	}
	return 0
}

// checkFindings is `relai validate`, plus the handoff budget, plus drift from the pinned hub.
func checkFindings(dir string, outdated bool) []validate.Finding {
	findings := validate.Run(dir)
	add := func(level, file, format string, a ...any) {
		findings = append(findings, validate.Finding{Level: level, File: file, Message: fmt.Sprintf(format, a...)})
	}

	max := handoff.DefaultOptions().MaxTokens
	if doc, err := handoff.Build(dir, handoff.DefaultOptions()); err != nil {
		add(validate.Warning, "handoff", "cannot be built here: %v", err)
	} else if n := handoff.EstimateTokens(doc); n > max {
		add(validate.Error, "handoff", "about %d tokens, the budget is %d: shorten .relai/notes.md", n, max)
	}

	if b, err := os.ReadFile(filepath.Join(dir, "AGENTS.md")); err == nil {
		for _, o := range hub.Overrides(string(b)) {
			add(validate.Info, "AGENTS.md", "declared override: %s", o)
		}
	}

	pin, ok, err := readPin(dir)
	switch {
	case err != nil:
		if _, statErr := os.Stat(filepath.Join(dir, "relai.yaml")); statErr == nil {
			add(validate.Error, "relai.yaml", "%v", err)
		}
		return findings
	case !ok:
		if outdated {
			add(validate.Info, "relai.yaml", "no conventions pinned: nothing to compare with the hub")
		}
		return findings
	}

	cache, err := hubCacheDir()
	var plan *hub.Plan
	if err == nil {
		var hubDir string
		if hubDir, err = hub.Fetch(pin, cache, false); err == nil {
			var content *hub.Content
			if content, err = hub.Load(hubDir); err == nil {
				plan, err = hub.BuildPlan(dir, pin, content)
			}
		}
	}
	if err != nil {
		add(validate.Error, "relai.yaml", "%v", err)
	} else {
		findings = append(findings, plan.Drift(pin)...)
	}

	if outdated {
		if tags, err := hub.ListTags(pin.Hub); err != nil {
			add(validate.Warning, "relai.yaml", "%v", err)
		} else if newest, found := hub.Newest(pin.Tag, tags); found {
			add(validate.Warning, "relai.yaml", "the hub has a newer version %s (pinned: %s): change the tag and run `relai sync --apply`", newest, pin.Tag)
		}
	}
	return findings
}

// cmdHub implements `relai hub init <dir>`.
func cmdHub(args []string, stdout, stderr io.Writer) int {
	if len(args) < 1 || args[0] != "init" {
		fmt.Fprintln(stderr, "usage: relai hub init <dir>")
		return 2
	}
	fs := newFlags("hub init", stderr)
	if code, stop := parseFlags(fs, args[1:]); stop {
		return code
	}
	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "usage: relai hub init <dir>")
		return 2
	}
	dir := fs.Arg(0)
	created, err := hub.InitHub(dir)
	for _, c := range created {
		fmt.Fprintln(stdout, "created ", c)
	}
	if err != nil {
		fmt.Fprintln(stderr, "relai:", err)
		return 1
	}
	fmt.Fprintf(stdout, "\nNext: git init, commit and tag it (git tag v0.1.0) in %s, then in each repository add\n  conventions: <path or clone URL of the hub>@v0.1.0\nto relai.yaml and run: relai sync --apply\n", dir)
	return 0
}
