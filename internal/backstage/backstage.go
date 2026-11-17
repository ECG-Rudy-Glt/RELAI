// Package backstage turns a relai.yaml into a Backstage catalog-info.yaml. It is a one-way export:
// Relai never reads Backstage and never depends on it, a platform team's catalog may consume the file.
//
// Mapping (see docs/backstage.md):
//
//	id          -> Component metadata.name (characters Backstage does not accept become "-")
//	summary     -> metadata.description
//	owners[0]   -> spec.owner (a bare name; Backstage reads it as a group)
//	provides    -> one API entity per capability, listed in spec.providesApis
//	consumes    -> spec.consumesApis (references to the API entities of the providers)
//	links       -> spec.dependsOn (components)
package backstage

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

// Options sets what relai.yaml does not say.
type Options struct {
	Type      string // Component spec.type, default "service"
	Lifecycle string // default "experimental"
	Namespace string // default "default"
}

func (o Options) withDefaults() Options {
	if o.Type == "" {
		o.Type = "service"
	}
	if o.Lifecycle == "" {
		o.Lifecycle = "experimental"
	}
	if o.Namespace == "" {
		o.Namespace = "default"
	}
	return o
}

var badChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// Name makes a Backstage entity name: [A-Za-z0-9._-], starts and ends with an alphanumeric, at most 63.
func Name(s string) string {
	n := badChars.ReplaceAllString(s, "-")
	n = strings.Trim(n, "-_.")
	if len(n) > 63 {
		n = strings.Trim(n[:63], "-_.")
	}
	if n == "" {
		n = "unnamed"
	}
	return n
}

func q(s string) string { return strconv.Quote(strings.Join(strings.Fields(s), " ")) }

// Render returns the multi-document YAML for one repository: its Component, then one API per capability
// it provides.
func Render(m index.Manifest, o Options) string {
	o = o.withDefaults()
	var b strings.Builder
	w := func(f string, a ...any) { fmt.Fprintf(&b, f+"\n", a...) }
	owner := "unknown"
	if len(m.Owners) > 0 {
		owner = Name(m.Owners[0])
	}
	var provides []string
	for _, c := range m.Provides {
		provides = append(provides, c.Capability)
	}
	sort.Strings(provides)
	w("apiVersion: backstage.io/v1alpha1")
	w("kind: Component")
	w("metadata:")
	w("  name: %s", q(Name(m.ID)))
	if m.ID != Name(m.ID) {
		w("  annotations:")
		w("    relai.dev/id: %s", q(m.ID))
	}
	if m.Summary != "" {
		w("  description: %s", q(m.Summary))
	}
	w("spec:")
	w("  type: %s", q(o.Type))
	w("  lifecycle: %s", q(o.Lifecycle))
	w("  owner: %s", q(owner))
	list := func(key string, items []string) {
		if len(items) == 0 {
			return
		}
		w("  %s:", key)
		for _, it := range items {
			w("    - %s", q(it))
		}
	}
	var apis []string
	for _, p := range provides {
		apis = append(apis, "api:"+o.Namespace+"/"+Name(p))
	}
	list("providesApis", apis)
	var cons []string
	for _, c := range m.Consumes {
		cons = append(cons, "api:"+o.Namespace+"/"+Name(c.Capability))
	}
	sort.Strings(cons)
	list("consumesApis", cons)
	var deps []string
	for _, l := range m.Links {
		deps = append(deps, "component:"+o.Namespace+"/"+Name(l))
	}
	list("dependsOn", deps)
	for _, p := range provides {
		w("---")
		w("apiVersion: backstage.io/v1alpha1")
		w("kind: API")
		w("metadata:")
		w("  name: %s", q(Name(p)))
		w("  description: %s", q("Capability "+p+" provided by "+m.ID+" (declared in relai.yaml)"))
		w("spec:")
		w("  type: %s", q("relai-capability"))
		w("  lifecycle: %s", q(o.Lifecycle))
		w("  owner: %s", q(owner))
		w("  definition: %s", q("capability: "+p))
	}
	return b.String()
}
