package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCopy_RefusesLinkToFileOutsideTheTemplate(t *testing.T) {
	root := t.TempDir()
	secret := filepath.Join(root, "secret.txt")
	writeFile(t, secret, "TOPSECRET")
	tpl := filepath.Join(root, "tpl")
	writeFile(t, filepath.Join(tpl, "a.txt"), "x")
	for name, target := range map[string]string{"abs": secret, "rel": "../secret.txt"} {
		if err := os.Symlink(target, filepath.Join(tpl, "creds-"+name)); err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join(root, "out-"+name)
		err := Copy(tpl, dest)
		if err == nil || !strings.Contains(err.Error(), "outside the template") {
			t.Fatalf("%s: Copy = %v, want a refusal", name, err)
		}
		if _, serr := os.Stat(dest); serr == nil {
			t.Errorf("%s: half-filled destination left behind", name)
		}
		os.Remove(filepath.Join(tpl, "creds-"+name))
	}
}

func TestCopy_KeepsLinksInsideTheTemplateAsLinks(t *testing.T) {
	root := t.TempDir()
	tpl := filepath.Join(root, "tpl")
	writeFile(t, filepath.Join(tpl, "docs", "real.md"), "hello")
	_ = os.Symlink("docs/real.md", filepath.Join(tpl, "link.md"))
	_ = os.Symlink("nowhere", filepath.Join(tpl, "broken")) // dangling but inside: harmless as a link
	dest := filepath.Join(root, "out")
	if err := Copy(tpl, dest); err != nil {
		t.Fatal(err)
	}
	if l, err := os.Readlink(filepath.Join(dest, "link.md")); err != nil || l != "docs/real.md" {
		t.Errorf("link not recreated as a link: %q, %v", l, err)
	}
	if _, err := os.Readlink(filepath.Join(dest, "broken")); err != nil {
		t.Errorf("dangling link not kept: %v", err)
	}
}

func TestCopy_RefusesDestinationInsideTemplate(t *testing.T) {
	tpl := t.TempDir()
	writeFile(t, filepath.Join(tpl, "a.txt"), "x")
	err := Copy(tpl, filepath.Join(tpl, "out"))
	if err == nil || !strings.Contains(err.Error(), "inside the template") {
		t.Fatalf("Copy = %v", err)
	}
	if _, err := os.Stat(filepath.Join(tpl, "out")); err == nil {
		t.Error("destination created inside the template")
	}
}

func TestNew_ReadOnlyTemplateFileIsSubstitutedAndKeepsItsMode(t *testing.T) {
	root := t.TempDir()
	tpl := filepath.Join(root, "tpl")
	writeFile(t, filepath.Join(tpl, "LICENSE"), "name {{.RepoName}}")
	if err := os.Chmod(filepath.Join(tpl, "LICENSE"), 0o444); err != nil {
		t.Fatal(err)
	}
	dest := filepath.Join(root, "widgets")
	if _, err := New(tpl, dest, Tokens{RepoName: "widgets"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dest, "LICENSE"))
	st, _ := os.Stat(filepath.Join(dest, "LICENSE"))
	if string(b) != "name widgets" || st.Mode().Perm() != 0o444 {
		t.Errorf("LICENSE = %q mode %v", b, st.Mode().Perm())
	}
}

func TestSubstitute_OnlyTheKnownTokenOtherBracesUntouched(t *testing.T) {
	dir := t.TempDir()
	ci := "name: {{.RepoName}}-ci\nsha: ${{ github.sha }}\nalso: {{ .RepoName }}\n"
	writeFile(t, filepath.Join(dir, "ci.yml"), ci)
	if err := Substitute(dir, Tokens{RepoName: "widgets"}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "ci.yml"))
	if want := "name: widgets-ci\nsha: ${{ github.sha }}\nalso: widgets\n"; string(b) != want {
		t.Errorf("ci.yml = %q, want %q", b, want)
	}
}

func TestSubstitute_BinaryAndHugeFilesAreByteForByte(t *testing.T) {
	dir := t.TempDir()
	bin := append([]byte("\x00\x01 {{.RepoName}} "), bytes.Repeat([]byte{0xff}, 100)...)
	_ = os.WriteFile(filepath.Join(dir, "x.bin"), bin, 0o644)
	big := strings.Repeat("{{.RepoName}}\n", 100000) // > 1 MiB
	_ = os.WriteFile(filepath.Join(dir, "big.txt"), []byte(big), 0o644)
	if err := Substitute(dir, Tokens{RepoName: "widgets"}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "x.bin")); !bytes.Equal(b, bin) {
		t.Error("binary file was modified")
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "big.txt")); string(b) != big {
		t.Error("oversized file was modified")
	}
}
