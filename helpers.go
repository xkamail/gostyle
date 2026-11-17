package main

import (
	"encoding/json"
	"go/ast"
	"go/types"
	"log/slog"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"
)

const (
	maxLines   = 100
	maxCallers = 2
)

type candidate struct {
	Func        string   `json:"func"`
	Pos         string   `json:"pos"`
	Lines       int      `json:"lines"`
	Exported    bool     `json:"exported"`
	Callers     []string `json:"callers"`
	TestCallers int      `json:"testCallers"`

	valueUse bool
	tests    map[string]bool
}

func runHelpers(patterns []string) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.NeedName | packages.NeedFiles | packages.NeedSyntax | packages.NeedTypes | packages.NeedTypesInfo,
		Tests: true,
	}, patterns...)
	if err != nil {
		slog.Error("load packages", "error", err)
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}
	wd, _ := os.Getwd()
	rel := func(p string) string {
		if r, err := filepath.Rel(wd, p); err == nil {
			return r
		}
		return p
	}

	// Without NeedDeps each package sees its imports through export data, so the
	// same func is a different *types.Func per importer; key by path.name instead.
	key := func(fn *types.Func) string { return fn.Pkg().Path() + "." + fn.Name() }
	cands := map[string]*candidate{}
	callers := map[string]map[string]bool{}

	for _, pkg := range pkgs {
		for _, f := range pkg.Syntax {
			name := pkg.Fset.File(f.Pos()).Name()
			if ast.IsGenerated(f) || strings.HasSuffix(name, "_test.go") {
				continue
			}
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Recv != nil || fd.Name.Name == "main" || fd.Name.Name == "init" ||
					strings.Contains(fd.Doc.Text(), "helperscan:keep") {
					continue
				}
				fn, ok := pkg.TypesInfo.Defs[fd.Name].(*types.Func)
				if !ok {
					continue
				}
				k := key(fn)
				if cands[k] != nil {
					continue
				}
				pos := pkg.Fset.Position(fd.Pos())
				cands[k] = &candidate{
					Func:     k,
					Pos:      rel(pos.Filename) + ":" + strconv.Itoa(pos.Line),
					Lines:    pkg.Fset.Position(fd.End()).Line - pos.Line + 1,
					Exported: fd.Name.IsExported(),
					tests:    map[string]bool{},
				}
			}
		}
	}

	for _, pkg := range pkgs {
		called := map[*ast.Ident]bool{}
		for _, f := range pkg.Syntax {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				fun := ast.Unparen(call.Fun)
				switch e := fun.(type) {
				case *ast.IndexExpr:
					fun = e.X
				case *ast.IndexListExpr:
					fun = e.X
				}
				switch e := fun.(type) {
				case *ast.Ident:
					called[e] = true
				case *ast.SelectorExpr:
					called[e.Sel] = true
				}
				return true
			})
		}
		for id, obj := range pkg.TypesInfo.Uses {
			fn, ok := obj.(*types.Func)
			if !ok || fn.Pkg() == nil {
				continue
			}
			c := cands[key(fn.Origin())]
			if c == nil {
				continue
			}
			pos := pkg.Fset.Position(id.Pos())
			at := rel(pos.Filename) + ":" + strconv.Itoa(pos.Line)
			switch {
			case !called[id]:
				c.valueUse = true
			case strings.HasSuffix(pos.Filename, "_test.go"):
				c.tests[at] = true
			default:
				if callers[c.Func] == nil {
					callers[c.Func] = map[string]bool{}
				}
				callers[c.Func][at] = true
			}
		}
	}

	out := []*candidate{}
	for k, c := range cands {
		c.Callers = slices.Sorted(maps.Keys(callers[k]))
		c.TestCallers = len(c.tests)
		if c.valueUse || c.Lines >= maxLines || len(c.Callers) > maxCallers {
			continue
		}
		out = append(out, c)
	}
	slices.SortFunc(out, func(a, b *candidate) int { return strings.Compare(a.Pos, b.Pos) })
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		slog.Error("encode", "error", err)
		os.Exit(1)
	}
}
