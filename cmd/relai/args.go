package main

import (
	"flag"
	"strings"
)

// reorderArgs moves flags in front of positional arguments, so "relai init dir --no-claude" means
// the same as "relai init --no-claude dir". Go's flag package stops parsing at the first
// positional argument and would otherwise treat the rest as plain arguments, silently ignoring the
// flag (and, for init, doing the opposite of what was asked). A "--" ends the flags, as usual.
// Unknown flags are left in place for fs.Parse to reject.
func reorderArgs(fs *flag.FlagSet, args []string) []string {
	boolean := map[string]bool{}
	fs.VisitAll(func(f *flag.Flag) {
		if b, ok := f.Value.(interface{ IsBoolFlag() bool }); ok && b.IsBoolFlag() {
			boolean[f.Name] = true
		}
	})
	var flags, rest []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			rest = append(rest, args[i+1:]...)
			break
		}
		if len(a) < 2 || a[0] != '-' {
			rest = append(rest, a)
			continue
		}
		flags = append(flags, a)
		name := strings.TrimLeft(a, "-")
		if strings.Contains(name, "=") || boolean[name] {
			continue
		}
		if i+1 < len(args) { // the flag's value
			i++
			flags = append(flags, args[i])
		}
	}
	if len(rest) > 0 && len(args) > 0 && containsDashDash(args) {
		return append(append(flags, "--"), rest...)
	}
	return append(flags, rest...)
}

func containsDashDash(args []string) bool {
	for _, a := range args {
		if a == "--" {
			return true
		}
	}
	return false
}
