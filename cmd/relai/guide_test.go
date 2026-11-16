package main

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func fakeHome(t *testing.T) func() (string, error) {
	h := t.TempDir()
	return func() (string, error) { return h, nil }
}

func TestRunGuide(t *testing.T) {
	cases := []struct {
		name string
		args []string
		code int
	}{
		{"print", nil, 0},
		{"unknown flag", []string{"--bogus"}, 2},
		{"force alone", []string{"--force"}, 2},
		{"extra argument", []string{"x"}, 2},
		{"install", []string{"--install"}, 0},
	}
	for _, c := range cases {
		var out, errb bytes.Buffer
		if code := runGuide(c.args, fakeHome(t), &out, &errb); code != c.code {
			t.Errorf("%s: code %d, want %d (%s)", c.name, code, c.code, errb.String())
		}
		if c.name == "print" && !strings.Contains(out.String(), "relai handoff") {
			t.Errorf("print: unexpected output %q", out.String())
		}
	}
}

func TestRunGuideInstallConflictExits1(t *testing.T) {
	home := fakeHome(t)
	var out, errb bytes.Buffer
	runGuide([]string{"--install"}, home, &out, &errb)
	h, _ := home()
	p := h + "/.claude/skills/relai/SKILL.md"
	os.WriteFile(p, []byte("mine"), 0o644)
	if code := runGuide([]string{"--install"}, home, &out, &errb); code != 1 {
		t.Errorf("code %d, want 1", code)
	}
	if code := runGuide([]string{"--install", "--force"}, home, &out, &errb); code != 0 {
		t.Errorf("force: code %d", code)
	}
	if code := runGuide([]string{"--install"}, func() (string, error) { return "", errors.New("no home") }, &out, &errb); code != 1 {
		t.Errorf("no home: code %d", code)
	}
}
