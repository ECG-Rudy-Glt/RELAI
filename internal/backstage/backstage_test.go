package backstage

import (
	"strings"
	"testing"

	"github.com/ECG-Rudy-Glt/RELAI/internal/index"
)

func TestRender(t *testing.T) {
	m := index.ParseManifest([]byte("id: org/invoice-pdf\nsummary: \"Makes PDFs.\"\nowners: [alice, team-billing]\nprovides:\n  - capability: pdf.generate.invoice\nconsumes:\n  - capability: auth.oidc\nlinks: [billing-api]\n"))
	doc := Render(m, Options{})
	for _, want := range []string{
		`name: "org-invoice-pdf"`, `relai.dev/id: "org/invoice-pdf"`, `description: "Makes PDFs."`, `owner: "alice"`,
		`- "api:default/pdf.generate.invoice"`, `consumesApis:`, `- "api:default/auth.oidc"`, `- "component:default/billing-api"`,
		"---\napiVersion: backstage.io/v1alpha1\nkind: API", `type: "relai-capability"`,
	} {
		if !strings.Contains(doc, want) {
			t.Errorf("missing %q in:\n%s", want, doc)
		}
	}
	if Render(m, Options{}) != doc {
		t.Error("not deterministic")
	}
}

func TestNameAndDefaults(t *testing.T) {
	for in, want := range map[string]string{"a/b c": "a-b-c", "-x-": "x", "": "unnamed", strings.Repeat("a", 80): strings.Repeat("a", 63)} {
		if got := Name(in); got != want {
			t.Errorf("Name(%q) = %q, want %q", in, got, want)
		}
	}
	doc := Render(index.Manifest{ID: "x"}, Options{Type: "library", Lifecycle: "production", Namespace: "team"})
	if !strings.Contains(doc, `type: "library"`) || !strings.Contains(doc, `lifecycle: "production"`) || !strings.Contains(doc, `owner: "unknown"`) {
		t.Errorf("options not applied:\n%s", doc)
	}
	if strings.Contains(Render(index.Manifest{ID: "x"}, Options{}), "providesApis") {
		t.Error("empty lists must be left out")
	}
}
