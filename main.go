// Command gostyle enforces and audits Go style for pgctx + arpc codebases.
//
//	gostyle lint ./...                   # analyzers; exits non-zero on findings
//	gostyle lint -acronym=false ./...    # disable one rule
//	gostyle helpers ./...                # small, rarely-called funcs as JSON (helper-review)
//	gostyle comments ./...               # narration-comment candidates as JSON (comment-review)
//	gostyle comments -lines ./...        # same, as file:line: message for reviewdog
//	gostyle skills install               # write the review skills into .claude/skills
//	gostyle version                      # print the gostyle version
//
// lint knows a package is a handler when it sits under <module>/api, /apis or /backoffice,
// and a worker under <module>/worker, where <module> is the analyzed package's own module.
// Repo-specific settings are flags: -moneydecimal.words / -moneydecimal.extra name the
// money fields, -validfirst.authfirst=<tree> lets an authorize* call precede Valid().
//
// helpers never fails: whether a helper earns its name is a judgment call. A func is
// skipped when its doc comment contains "helperscan:keep", when it is referenced as a value,
// or when it lives in a generated or _test.go file.
//
// comments never fails either. It scans comments inside func bodies and doc comments on
// unexported funcs; exported doc comments and struct field comments are left alone because
// godoc and generated OpenAPI specs read them. The unambiguous cases (banners, multi-line
// commented-out code) are lint's slopcomment rule.
package main

import (
	"fmt"
	"os"
	"runtime/debug"
)

const usage = "usage: gostyle <lint|helpers|comments|skills|version> [flags] <packages>"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	switch cmd {
	case "lint":
		os.Args = append([]string{"gostyle lint"}, args...)
		runLint()
	case "helpers":
		runHelpers(args)
	case "comments":
		runComments(args)
	case "skills":
		runSkills(args)
	case "version":
		fmt.Println(version())
	default:
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(2)
	}
}

// version is the module version Go stamped into the binary: vX.Y.Z when run as
// `go tool gostyle` or installed with `go install ...@vX.Y.Z`, "(devel)" from a checkout.
func version() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "(unknown)"
	}
	return bi.Main.Version
}
