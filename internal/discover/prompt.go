package discover

import (
	"fmt"
	"strings"
)

// Prompt returns instructions for the user's own agent to draft .relai/notes.md from the sources
// found in inv. Relai never calls an LLM itself: the agent writes the draft, a person reviews it.
func Prompt(inv Inventory) string {
	var b strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	p("Draft `.relai/notes.md` for this repository so that the next person or agent who arrives cold")
	p("knows the decisions, the pitfalls and, if no task tracker says it, the next step.")
	p("")
	p("Sources found by `relai discover` (read only what you need, newest entries first):")
	n := 0
	for _, kind := range []string{Instructions, Docs, Decisions, Tasks, Specs} {
		if g := inv.Get(kind); g != nil {
			p("- %s: %s", kind, strings.Join(g.Items, ", "))
			n++
		}
	}
	if n == 0 {
		p("- none: use the README and `git log` only.")
	}
	p("")
	p("Write the file with exactly these sections, one line per entry, at most 40 lines in total:")
	p("")
	p("    ### Next")
	p("    - <the next step, only if no task tracker or spec gives it; omit the section otherwise>")
	p("    ### Decisions")
	p("    - <YYYY-MM-DD> — <decision> — <why>")
	p("    ### Pitfalls")
	p("    - <symptom> → <cause> → <what to do>")
	p("")
	p("Rules:")
	p("- Only facts written in the sources; cite the source path in brackets at the end of each line.")
	p("- Keep decisions that still hold and pitfalls that will bite again; drop history.")
	p("- No secrets, tokens, passwords or internal hostnames.")
	p("- If `.relai/notes.md` already exists, keep its lines and add only what is missing.")
	p("- Do not modify any other file. Show the result to the user for review before committing it.")
	return b.String()
}
