package main

import (
	"flag"
	"fmt"
	"io"

	"github.com/ECG-Rudy-Glt/RELAI/internal/guide"
)

// runGuide implements `relai guide [--install [--force]]` and returns the exit code.
func runGuide(args []string, home func() (string, error), stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("guide", flag.ContinueOnError)
	fs.SetOutput(stderr)
	install := fs.Bool("install", false, "write the guide as an Agent Skill in ~/.claude/skills/relai and ~/.agents/skills/relai")
	force := fs.Bool("force", false, "with --install: overwrite files that differ")
	if err := fs.Parse(reorderArgs(fs, args)); err != nil {
		return 2
	}
	if fs.NArg() > 0 || (*force && !*install) {
		fmt.Fprintln(stderr, "usage: relai guide [--install [--force]]")
		return 2
	}
	if !*install {
		fmt.Fprint(stdout, guide.Text())
		return 0
	}
	h, err := home()
	if err != nil {
		fmt.Fprintln(stderr, "relai:", err)
		return 1
	}
	rs, err := guide.Install(h, *force)
	for _, r := range rs {
		fmt.Fprintf(stdout, "%-20s %s\n", r.Status, r.Path)
	}
	if err != nil {
		fmt.Fprintln(stderr, "relai:", err)
		return 1
	}
	return 0
}
