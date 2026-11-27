package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

type coverTest struct {
	Test    string   `json:"test"`
	Pos     string   `json:"pos"`
	Blocks  int      `json:"blocks"`
	Unique  []string `json:"unique"`
	Closest string   `json:"closest,omitempty"`
	Shared  int      `json:"shared"`
	Seconds float64  `json:"seconds"`
	Failed  bool     `json:"failed,omitempty"`

	covered map[string]bool
}

func runTestCover(patterns []string) {
	out, err := testCover("", patterns)
	if err != nil {
		slog.Error("testcover", "error", err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		slog.Error("encode", "error", err)
		os.Exit(1)
	}
}

// testCover runs each top-level test of each package alone, with coverage of that package
// only, and reports per test the blocks no other test of the package reaches.
func testCover(dir string, patterns []string) ([]*coverTest, error) {
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}
	list := exec.Command("go", append([]string{"list", "-f", "{{if or .TestGoFiles .XTestGoFiles}}{{.ImportPath}}\t{{.Dir}}{{end}}"}, patterns...)...)
	list.Dir = dir
	list.Stderr = os.Stderr
	listed, err := list.Output()
	if err != nil {
		return nil, fmt.Errorf("go list: %w", err)
	}
	tmp, err := os.MkdirTemp("", "gostyle-testcover")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmp)

	wd, _ := os.Getwd()
	out := []*coverTest{}
	for line := range strings.Lines(string(listed)) {
		pkg, pkgDir, ok := strings.Cut(strings.TrimSpace(line), "\t")
		if !ok {
			continue
		}
		rel := func(file string) string {
			p := filepath.Join(pkgDir, file)
			if r, err := filepath.Rel(wd, p); err == nil {
				return r
			}
			return p
		}
		bin := filepath.Join(tmp, "pkg.test")
		build := exec.Command("go", "test", "-c", "-cover", "-covermode=set", "-coverpkg="+pkg, "-o", bin, pkg)
		build.Dir = dir
		build.Stderr = os.Stderr
		if err := build.Run(); err != nil {
			slog.Error("build test binary", "pkg", pkg, "error", err)
			continue
		}
		names, err := exec.Command(bin, "-test.list", ".").Output()
		if err != nil {
			slog.Error("list tests", "pkg", pkg, "error", err)
			continue
		}
		pos := testPositions(pkgDir)
		var tests []*coverTest
		for name := range strings.Lines(string(names)) {
			name = strings.TrimSpace(name)
			if !isTestFunc(name) {
				continue
			}
			slog.Info("run", "pkg", pkg, "test", name)
			t := &coverTest{Test: name, Pos: rel(pos[name])}
			profile := filepath.Join(tmp, "cover.out")
			run := exec.Command(bin, "-test.run", "^"+name+"$", "-test.count=1", "-test.coverprofile="+profile)
			run.Dir = pkgDir
			start := time.Now()
			err := run.Run()
			t.Seconds = time.Since(start).Round(10 * time.Millisecond).Seconds()
			if err != nil {
				t.Failed = true
			} else if t.covered, err = coveredBlocks(profile); err != nil {
				return nil, err
			}
			tests = append(tests, t)
		}
		for _, t := range tests {
			t.Blocks = len(t.covered)
			t.Unique = []string{}
			for b := range t.covered {
				if !slices.ContainsFunc(tests, func(o *coverTest) bool { return o != t && o.covered[b] }) {
					t.Unique = append(t.Unique, rel(b))
				}
			}
			slices.Sort(t.Unique)
			for _, o := range tests {
				var shared int
				for b := range t.covered {
					if o.covered[b] {
						shared++
					}
				}
				if o != t && shared > t.Shared {
					t.Closest, t.Shared = o.Test, shared
				}
			}
		}
		out = append(out, tests...)
	}
	return out, nil
}

// coveredBlocks reads a set-mode cover profile into the blocks it reached,
// keyed by base file name and line range, e.g. "abs.go:5-7".
func coveredBlocks(profile string) (map[string]bool, error) {
	data, err := os.ReadFile(profile)
	if err != nil {
		return nil, err
	}
	covered := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		// path/file.go:startLine.startCol,endLine.endCol numStmt count
		loc, rest, ok := strings.Cut(sc.Text(), " ")
		if !ok || strings.HasPrefix(loc, "mode:") || strings.HasSuffix(rest, " 0") {
			continue
		}
		file, span, _ := strings.Cut(loc, ":")
		start, end, _ := strings.Cut(span, ",")
		startLine, _, _ := strings.Cut(start, ".")
		endLine, _, _ := strings.Cut(end, ".")
		covered[filepath.Base(file)+":"+startLine+"-"+endLine] = true
	}
	return covered, sc.Err()
}

// testPositions maps each top-level test func in dir's _test.go files to file:line.
func testPositions(dir string) map[string]string {
	pos := map[string]string{}
	files, _ := filepath.Glob(filepath.Join(dir, "*_test.go"))
	fset := token.NewFileSet()
	for _, path := range files {
		f, err := parser.ParseFile(fset, path, nil, parser.SkipObjectResolution)
		if err != nil {
			continue
		}
		for _, d := range f.Decls {
			if fd, ok := d.(*ast.FuncDecl); ok && fd.Recv == nil && isTestFunc(fd.Name.Name) {
				pos[fd.Name.Name] = filepath.Base(path) + ":" + strconv.Itoa(fset.Position(fd.Pos()).Line)
			}
		}
	}
	return pos
}
