package main

import (
	"flag"
	"reflect"
	"testing"
)

func TestReorderArgs(t *testing.T) {
	newFS := func() *flag.FlagSet {
		fs := flag.NewFlagSet("t", flag.ContinueOnError)
		fs.Bool("no-claude", false, "")
		fs.String("o", "", "")
		fs.String("index", "", "")
		return fs
	}
	cases := []struct {
		in       []string
		wantFlag map[string]string
		wantArgs []string
		noClaude bool
	}{
		{[]string{"dir", "--no-claude"}, nil, []string{"dir"}, true},
		{[]string{"repos.txt", "-o", "after.json"}, map[string]string{"o": "after.json"}, []string{"repos.txt"}, false},
		{[]string{"auth.oidc", "--index=x.json"}, map[string]string{"index": "x.json"}, []string{"auth.oidc"}, false},
		{[]string{"--no-claude", "dir"}, nil, []string{"dir"}, true},
		{[]string{"a", "-o", "o.json", "b"}, map[string]string{"o": "o.json"}, []string{"a", "b"}, false},
		{[]string{"--", "-o"}, nil, []string{"-o"}, false},
	}
	for _, c := range cases {
		fs := newFS()
		if err := fs.Parse(reorderArgs(fs, c.in)); err != nil {
			t.Fatalf("%v: %v", c.in, err)
		}
		if !reflect.DeepEqual(append([]string{}, fs.Args()...), c.wantArgs) {
			t.Errorf("%v: args = %v, want %v", c.in, fs.Args(), c.wantArgs)
		}
		for k, v := range c.wantFlag {
			if fs.Lookup(k).Value.String() != v {
				t.Errorf("%v: -%s = %q, want %q", c.in, k, fs.Lookup(k).Value, v)
			}
		}
		if got := fs.Lookup("no-claude").Value.String() == "true"; got != c.noClaude {
			t.Errorf("%v: no-claude = %v", c.in, got)
		}
	}
}
