package main

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

func TestScanFile(t *testing.T) {
	src := `package p

// Load reads the config.
func Load() {}

// helper returns x.
func helper(x int) int { return x }

func f(userID string) error {
	// create user
	createUser(userID)
	// retry once: the gateway drops the first request after a cold start
	send()
	// user id
	lookup(userID)
	//req.Header.Set("X", y)
	return nil
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "p.go", src, parser.ParseComments)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, fd := range scanFile(fset, f, strings.Split(src, "\n"), "p.go") {
		got[fd.Pos] = fd.Reason
	}
	want := map[string]string{
		"p.go:6":  "restates-name",
		"p.go:10": "narration",
		"p.go:14": "restates-code",
		"p.go:16": "commented-out-code",
	}
	if len(got) != len(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	for pos, reason := range want {
		if got[pos] != reason {
			t.Errorf("%s: reason = %q, want %q", pos, got[pos], reason)
		}
	}
}
