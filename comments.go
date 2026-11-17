package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/tools/go/packages"
)

type finding struct {
	Pos    string `json:"pos"`
	Reason string `json:"reason"`
	Text   string `json:"text"`
	Code   string `json:"code"`
}

var (
	narrationVerbs = map[string]bool{
		"add": true, "build": true, "calculate": true, "call": true, "check": true, "compute": true,
		"convert": true, "create": true, "define": true, "delete": true, "fetch": true, "get": true,
		"handle": true, "increment": true, "initialize": true, "insert": true, "iterate": true,
		"load": true, "loop": true, "parse": true, "return": true, "save": true, "send": true,
		"set": true, "update": true, "validate": true,
	}
	stopWords = map[string]bool{
		"a": true, "an": true, "and": true, "by": true, "for": true, "from": true, "if": true,
		"in": true, "into": true, "is": true, "it": true, "of": true, "on": true, "or": true,
		"the": true, "this": true, "to": true, "we": true, "with": true,
	}
	whyMarker = regexp.MustCompile(`(?i)\b(because|since|otherwise|so that|must|never|don't|can't|won't|instead|unless|gotcha|workaround)\b|[:;]|—|e\.g\.`)
	wordRe    = regexp.MustCompile(`[A-Za-z]+`)
	camelRe   = regexp.MustCompile(`[A-Z]+[a-z]*|[a-z]+`)
)

func runComments(args []string) {
	fs := flag.NewFlagSet("gostyle comments", flag.ExitOnError)
	lines := fs.Bool("lines", false, "print file:line:col: message instead of JSON")
	_ = fs.Parse(args)
	cfg := &packages.Config{Mode: packages.NeedName | packages.NeedFiles | packages.NeedSyntax}
	pkgs, err := packages.Load(cfg, fs.Args()...)
	if err != nil {
		slog.Error("load packages", "error", err)
		os.Exit(1)
	}
	wd, _ := os.Getwd()
	out := []finding{}
	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			name := pkg.Fset.File(f.Pos()).Name()
			if ast.IsGenerated(f) || strings.HasSuffix(name, "_test.go") {
				continue
			}
			src, err := os.ReadFile(name)
			if err != nil {
				slog.Error("read file", "file", name, "error", err)
				os.Exit(1)
			}
			rel, _ := filepath.Rel(wd, name)
			out = append(out, scanFile(pkg.Fset, f, strings.Split(string(src), "\n"), rel)...)
		}
	}
	if *lines {
		for _, fd := range out {
			first, _, _ := strings.Cut(fd.Text, "\n")
			fmt.Printf("%s:1: %s comment %q: delete it, or rewrite it to say why (see /comment-review)\n", fd.Pos, fd.Reason, first)
		}
		return
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		slog.Error("encode", "error", err)
		os.Exit(1)
	}
}

func scanFile(fset *token.FileSet, f *ast.File, lines []string, rel string) []finding {
	var bodies []*ast.BlockStmt
	unexportedDocs := map[*ast.CommentGroup]string{}
	ast.Inspect(f, func(n ast.Node) bool {
		switch n := n.(type) {
		case *ast.FuncDecl:
			if n.Body != nil {
				bodies = append(bodies, n.Body)
			}
			if n.Doc != nil && !n.Name.IsExported() {
				unexportedDocs[n.Doc] = n.Name.Name
			}
		case *ast.FuncLit:
			bodies = append(bodies, n.Body)
		}
		return true
	})
	inBody := func(g *ast.CommentGroup) bool {
		for _, b := range bodies {
			if b.Pos() < g.Pos() && g.End() < b.End() {
				return true
			}
		}
		return false
	}

	var out []finding
	for _, g := range f.Comments {
		text := strings.TrimSpace(g.Text())
		if text == "" || strings.HasPrefix(g.List[0].Text, "/*") {
			continue
		}
		pos := fset.Position(g.Pos())
		add := func(reason, code string) {
			out = append(out, finding{Pos: rel + ":" + strconv.Itoa(pos.Line), Reason: reason, Text: text, Code: code})
		}
		if fn, ok := unexportedDocs[g]; ok {
			if strings.HasPrefix(text, fn+" ") && !strings.Contains(text, "\n") && !whyMarker.MatchString(text) {
				add("restates-name", strings.TrimSpace(lines[fset.Position(g.End()).Line]))
			}
			continue
		}
		if !inBody(g) {
			continue
		}
		code := codeFor(lines, pos, fset.Position(g.End()).Line)
		switch {
		case !strings.Contains(text, "\n") && codeToken.MatchString(text) && parsesAsGo(text):
			add("commented-out-code", code)
		case whyMarker.MatchString(text):
		case narrationVerbs[strings.ToLower(firstWord(text))]:
			add("narration", code)
		case restates(text, code):
			add("restates-code", code)
		}
	}
	return out
}

// codeFor returns the code a comment describes: the code before it on its own line for a
// trailing comment, else the next non-blank, non-comment line.
func codeFor(lines []string, start token.Position, endLine int) string {
	if before := strings.TrimSpace(lines[start.Line-1][:start.Column-1]); before != "" {
		return before
	}
	for _, l := range lines[endLine:] {
		if l = strings.TrimSpace(l); l != "" && !strings.HasPrefix(l, "//") {
			return l
		}
	}
	return ""
}

// restates reports whether most content words of a short comment appear among the
// camelCase-split identifiers of the code it sits on.
func restates(text, code string) bool {
	codeWords := map[string]bool{}
	for _, w := range camelRe.FindAllString(code, -1) {
		codeWords[strings.ToLower(w)] = true
	}
	var words, hits int
	for _, w := range wordRe.FindAllString(text, -1) {
		w = strings.ToLower(w)
		if stopWords[w] {
			continue
		}
		words++
		if codeWords[w] || codeWords[strings.TrimSuffix(w, "s")] {
			hits++
		}
	}
	return words > 0 && words <= 8 && hits*2 >= words
}

func parsesAsGo(text string) bool {
	_, err := parser.ParseFile(token.NewFileSet(), "", "package p\nfunc _() {\n"+text+"\n}", 0)
	return err == nil
}

func firstWord(s string) string {
	end := strings.IndexFunc(s, func(r rune) bool { return !unicode.IsLetter(r) })
	if end < 0 {
		return s
	}
	return s[:end]
}
