package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

// writeFind prints who provides (or, with consumers, who consumes) capability. Finding nothing is not an
// error: it is stated and the caller exits 0, as `relai find` always has.
func writeFind(w io.Writer, entries []index.Entry, capability string, consumers, asJSON bool, me index.Manifest) error {
	mode, verb, found := "providers", "provides", index.Find(entries, capability)
	if consumers {
		mode, verb, found = "consumers", "consumes", index.FindConsumers(entries, capability)
	}
	found = index.Order(found, me)
	if asJSON {
		if found == nil {
			found = []index.Entry{}
		}
		b, err := json.MarshalIndent(struct {
			Capability string        `json:"capability"`
			Mode       string        `json:"mode"`
			Matches    []index.Entry `json:"matches"`
		}{capability, mode, found}, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(w, string(b))
		return err
	}
	if len(found) == 0 {
		fmt.Fprintf(w, "nothing in the index %s %q", verb, capability)
		if consumers {
			fmt.Fprint(w, " (the index only knows what repositories declare under consumes:)")
		}
		fmt.Fprintln(w)
		return nil
	}
	if consumers {
		fmt.Fprintf(w, "repos in the index that consume %q:\n", capability)
	}
	for _, e := range found {
		rel := index.Relation(e, me)
		if rel != "" {
			rel = " [" + rel + "]"
		}
		fmt.Fprintf(w, "- %s (%s)%s: %s\n", e.RelaiYAML.ID, e.Source, rel, e.RelaiYAML.Summary)
		if len(e.RelaiYAML.Links) > 0 {
			fmt.Fprintf(w, "    works with: %s\n", strings.Join(e.RelaiYAML.Links, ", "))
		}
	}
	return nil
}
