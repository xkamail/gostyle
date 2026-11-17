package main

import (
	"go/ast"
	"go/constant"
	"go/parser"
	"go/token"
	"go/types"
	"reflect"
	"regexp"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
	"golang.org/x/tools/go/types/typeutil"
)

const (
	pgctxPath = "github.com/acoshift/pgsql/pgctx"
)

var (
	upperSQL  = regexp.MustCompile(`\b(SELECT|INSERT|UPDATE|DELETE|FROM|WHERE|JOIN|AND|OR|NOT|ORDER|GROUP|BY|HAVING|VALUES|SET|INTO|RETURNING|LIMIT|OFFSET|EXISTS|AS|ON|CONFLICT|DO|NOTHING|IN|IS|NULL|CASE|WHEN|THEN|ELSE|END|DISTINCT|COUNT|COALESCE|SUM)\b`)
	sqlQuoted = regexp.MustCompile(`'[^']*'|"[^"]*"`)
	acronym   = regexp.MustCompile(`(^|[a-z0-9])(Id|Ids|Url|Urls|Uri|Api|Http|Json|Sql|Ip)([A-Z0-9]|$)`)
	codeToken = regexp.MustCompile(`[(){}]|:=|\breturn\b`)
	banner    = regexp.MustCompile(`^\s*[-=*#~/]{3,}`)
)

func runLint() {
	multichecker.Main(
		pgIter, sqlStyle, discardedErr, handlerErr, stdLog, ctxBackground,
		acronymName, moneyDecimal, validFirst, omitEmpty, slopComment, zeroDecl,
	)
}

func pgFunc(pass *analysis.Pass, call *ast.CallExpr) string {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != pgctxPath {
		return ""
	}
	return fn.Name()
}

// modulePath is the analyzed package's module, so the /api, /backoffice and /worker
// trees resolve in whichever repo gostyle runs in.
func modulePath(pass *analysis.Pass) string {
	if pass.Module == nil {
		return ""
	}
	return pass.Module.Path
}

func isTest(pass *analysis.Pass, n ast.Node) bool {
	return strings.HasSuffix(pass.Fset.File(n.Pos()).Name(), "_test.go")
}

func isHandlerPkg(pass *analysis.Pass) bool {
	p, module := pass.Pkg.Path(), modulePath(pass)
	for _, tree := range []string{"/api", "/apis", "/backoffice"} {
		if p == module+tree || strings.HasPrefix(p, module+tree+"/") {
			return true
		}
	}
	return false
}

func handlerSig(pass *analysis.Pass, fd *ast.FuncDecl) (paramName string, params *types.Named, result types.Type, ok bool) {
	if fd.Recv != nil || fd.Body == nil || !fd.Name.IsExported() {
		return "", nil, nil, false
	}
	sig, ok := pass.TypesInfo.Defs[fd.Name].Type().(*types.Signature)
	if !ok || sig.Params().Len() != 2 || sig.Results().Len() != 2 {
		return "", nil, nil, false
	}
	pt := sig.Params().At(1).Type()
	if ptr, ok := pt.(*types.Pointer); ok {
		pt = ptr.Elem()
	}
	named, ok := pt.(*types.Named)
	if !ok || !strings.HasSuffix(named.Obj().Name(), "Params") {
		return "", nil, nil, false
	}
	return sig.Params().At(1).Name(), named, sig.Results().At(0).Type(), true
}

var pgIter = &analysis.Analyzer{
	Name: "pgiter",
	Doc:  "use pgctx.Iter instead of pgctx.Query + rows.Next",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, f := range pass.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				if call, ok := n.(*ast.CallExpr); ok && pgFunc(pass, call) == "Query" {
					pass.Reportf(call.Pos(), "use pgctx.Iter instead of pgctx.Query + rows.Next")
				}
				return true
			})
		}
		return nil, nil
	},
}

var sqlStyle = &analysis.Analyzer{
	Name: "sqlstyle",
	Doc:  "SQL passed to pgctx must be a lowercase backtick string",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, f := range pass.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				idx := 1
				switch pgFunc(pass, call) {
				case "Exec", "QueryRow", "Query":
				case "Iter":
					idx = 2
				default:
					return true
				}
				if len(call.Args) <= idx {
					return true
				}
				lit, ok := call.Args[idx].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					return true
				}
				if lit.Value[0] == '"' {
					pass.Reportf(lit.Pos(), "SQL must be a backtick string, not double-quoted")
				}
				if m := upperSQL.FindString(sqlQuoted.ReplaceAllString(lit.Value, "")); m != "" {
					pass.Reportf(lit.Pos(), "SQL keywords must be lowercase (found %q)", m)
				}
				return true
			})
		}
		return nil, nil
	},
}

var discardedErr = &analysis.Analyzer{
	Name: "discardederr",
	Doc:  "errors from pgctx and json.Unmarshal must be handled",
	Run: func(pass *analysis.Pass) (any, error) {
		mustCheck := func(e ast.Expr) bool {
			call, ok := e.(*ast.CallExpr)
			if !ok {
				return false
			}
			if pgFunc(pass, call) != "" {
				errType := types.Universe.Lookup("error").Type()
				t := pass.TypesInfo.TypeOf(call)
				if tup, ok := t.(*types.Tuple); ok {
					for v := range tup.Variables() {
						if types.Identical(v.Type(), errType) {
							return true
						}
					}
					return false
				}
				return t != nil && types.Identical(t, errType)
			}
			if fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func); ok && fn.Pkg() != nil &&
				fn.Pkg().Path() == "encoding/json" && fn.Name() == "Unmarshal" {
				return true
			}
			if sel, ok := call.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Scan" {
				if inner, ok := sel.X.(*ast.CallExpr); ok && pgFunc(pass, inner) == "QueryRow" {
					return true
				}
			}
			return false
		}
		for _, f := range pass.Files {
			ast.Inspect(f, func(n ast.Node) bool {
				switch s := n.(type) {
				case *ast.ExprStmt:
					if mustCheck(s.X) {
						pass.Reportf(s.Pos(), "error is ignored; handle it")
					}
				case *ast.AssignStmt:
					for _, l := range s.Lhs {
						if id, ok := l.(*ast.Ident); !ok || id.Name != "_" {
							return true
						}
					}
					if len(s.Rhs) == 1 && mustCheck(s.Rhs[0]) {
						pass.Reportf(s.Pos(), "error is discarded with _; handle it")
					}
				}
				return true
			})
		}
		return nil, nil
	},
}

var handlerErr = &analysis.Analyzer{
	Name: "handlererr",
	Doc:  "client-facing handler errors must use arpc.NewError / arpc.NewErrorCode",
	Run: func(pass *analysis.Pass) (any, error) {
		if !isHandlerPkg(pass) {
			return nil, nil
		}
		for _, f := range pass.Files {
			if isTest(pass, f) {
				continue
			}
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok || fd.Body == nil || !fd.Name.IsExported() || fd.Type.Results == nil {
					continue
				}
				ast.Inspect(fd.Body, func(n ast.Node) bool {
					if _, ok := n.(*ast.FuncLit); ok {
						return false
					}
					ret, ok := n.(*ast.ReturnStmt)
					if !ok || len(ret.Results) == 0 {
						return true
					}
					call, ok := ret.Results[len(ret.Results)-1].(*ast.CallExpr)
					if !ok || len(call.Args) == 0 {
						return true
					}
					fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
					if !ok || fn.Pkg() == nil {
						return true
					}
					name := fn.Pkg().Path() + "." + fn.Name()
					if name != "errors.New" && name != "fmt.Errorf" {
						return true
					}
					if lit, ok := call.Args[0].(*ast.BasicLit); ok && !strings.Contains(lit.Value, "%w") {
						pass.Reportf(call.Pos(), "%s reaches the client as \"internal error\"; use arpc.NewError / arpc.NewErrorCode", name)
					}
					return true
				})
			}
		}
		return nil, nil
	},
}

var stdLog = &analysis.Analyzer{
	Name: "stdlog",
	Doc:  "use log/slog, not the standard log package",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, f := range pass.Files {
			for _, imp := range f.Imports {
				if imp.Path.Value == `"log"` && !isTest(pass, f) {
					pass.Reportf(imp.Pos(), "use log/slog instead of log")
				}
			}
		}
		return nil, nil
	},
}

var ctxBackground = &analysis.Analyzer{
	Name: "ctxbackground",
	Doc:  "handlers and workers must derive context from the request/task (pgctx lives on it)",
	Run: func(pass *analysis.Pass) (any, error) {
		if !isHandlerPkg(pass) && pass.Pkg.Path() != modulePath(pass)+"/worker" {
			return nil, nil
		}
		for _, f := range pass.Files {
			if isTest(pass, f) {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok {
					return true
				}
				if fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func); ok && fn.Pkg() != nil &&
					fn.Pkg().Path() == "context" && (fn.Name() == "Background" || fn.Name() == "TODO") {
					pass.Reportf(call.Pos(), "context.%s drops the request context (and its DB); use context.WithoutCancel(ctx)", fn.Name())
				}
				return true
			})
		}
		return nil, nil
	},
}

var acronymName = &analysis.Analyzer{
	Name: "acronym",
	Doc:  "acronyms in identifiers are all caps: ID, URL, API, HTTP, JSON, SQL, IP",
	Run: func(pass *analysis.Pass) (any, error) {
		for id, obj := range pass.TypesInfo.Defs {
			if obj == nil || id.Name == "_" {
				continue
			}
			if m := acronym.FindStringSubmatch(id.Name); m != nil {
				pass.Reportf(id.Pos(), "%s: write %q as %q", id.Name, m[2], strings.ToUpper(m[2]))
			}
		}
		return nil, nil
	},
}

var (
	moneyWords     = "price,amount,balance,quantity,fee"
	moneyExtra     string
	authorizeFirst string
)

func init() {
	moneyDecimal.Flags.StringVar(&moneyWords, "words", moneyWords, "comma-separated name stems that hold money")
	moneyDecimal.Flags.StringVar(&moneyExtra, "extra", "", "comma-separated stems added to -words")
	validFirst.Flags.StringVar(&authorizeFirst, "authfirst", "", "handler tree (e.g. backoffice) where an authorize* call may precede Valid()")
}

// moneyRegexp matches a name whose word starts with one of the stems, in camelCase
// or snake_case, so "totalAmount" and "fee_total" match but "management" does not.
func moneyRegexp(words string) *regexp.Regexp {
	var lower, upper []string
	for w := range strings.SplitSeq(words, ",") {
		if w = strings.ToLower(strings.TrimSpace(w)); w != "" {
			lower = append(lower, regexp.QuoteMeta(w))
			upper = append(upper, regexp.QuoteMeta(strings.ToUpper(w[:1])+w[1:]))
		}
	}
	return regexp.MustCompile(`(^(` + strings.Join(lower, "|") + `)|(` + strings.Join(upper, "|") + `))s?([A-Z0-9_]|$)`)
}

var moneyDecimal = &analysis.Analyzer{
	Name: "moneydecimal",
	Doc:  "money fields and params (names from -words/-extra) are decimal.Decimal, never float",
	Run: func(pass *analysis.Pass) (any, error) {
		moneyName := moneyRegexp(moneyWords + "," + moneyExtra)
		for id, obj := range pass.TypesInfo.Defs {
			v, ok := obj.(*types.Var)
			if !ok || v.Kind() == types.LocalVar || v.Kind() == types.RecvVar ||
				isTest(pass, id) || !moneyName.MatchString(id.Name) {
				continue
			}
			t := v.Type()
			if ptr, ok := t.(*types.Pointer); ok {
				t = ptr.Elem()
			}
			if b, ok := t.Underlying().(*types.Basic); ok && b.Info()&types.IsFloat != 0 {
				pass.Reportf(id.Pos(), "%s is %s; use decimal.Decimal (github.com/shopspring/decimal)", id.Name, v.Type())
			}
		}
		return nil, nil
	},
}

var validFirst = &analysis.Analyzer{
	Name: "validfirst",
	Doc:  "handlers validate params first: if err := p.Valid(); err != nil (after an authorize* call in the -authfirst tree)",
	Run: func(pass *analysis.Pass) (any, error) {
		if !isHandlerPkg(pass) {
			return nil, nil
		}
		authTree := authorizeFirst != "" && strings.HasPrefix(pass.Pkg.Path()+"/", modulePath(pass)+"/"+authorizeFirst+"/")
		initCall := func(s ast.Stmt) *ast.CallExpr {
			ifs, ok := s.(*ast.IfStmt)
			if !ok {
				return nil
			}
			as, ok := ifs.Init.(*ast.AssignStmt)
			if !ok || len(as.Rhs) != 1 {
				return nil
			}
			call, _ := as.Rhs[0].(*ast.CallExpr)
			return call
		}
		for _, f := range pass.Files {
			if isTest(pass, f) {
				continue
			}
			for _, d := range f.Decls {
				fd, ok := d.(*ast.FuncDecl)
				if !ok {
					continue
				}
				p, params, _, ok := handlerSig(pass, fd)
				if !ok || p == "" || p == "_" || len(fd.Body.List) == 0 {
					continue
				}
				if m, _, _ := types.LookupFieldOrMethod(types.NewPointer(params), true, params.Obj().Pkg(), "Valid"); m == nil {
					continue
				}
				stmts := fd.Body.List
				_, onlyReturn := stmts[0].(*ast.ReturnStmt)
				if delegatesToCallee := onlyReturn && len(stmts) == 1; delegatesToCallee {
					continue
				}
				isNilDefault := func(s ast.Stmt) bool {
					ifs, ok := s.(*ast.IfStmt)
					if !ok || ifs.Init != nil {
						return false
					}
					be, ok := ifs.Cond.(*ast.BinaryExpr)
					if !ok || be.Op != token.EQL {
						return false
					}
					x, ok := be.X.(*ast.Ident)
					return ok && x.Name == p
				}
				isAuthorizeCall := func(s ast.Stmt) bool {
					c := initCall(s)
					if !authTree || c == nil {
						return false
					}
					fn, ok := typeutil.Callee(pass.TypesInfo, c).(*types.Func)
					return ok && strings.HasPrefix(strings.ToLower(fn.Name()), "authorize")
				}
				for len(stmts) > 0 && (isNilDefault(stmts[0]) || isAuthorizeCall(stmts[0])) {
					stmts = stmts[1:]
				}
				if len(stmts) > 0 {
					if c := initCall(stmts[0]); c != nil {
						if sel, ok := c.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "Valid" {
							if x, ok := sel.X.(*ast.Ident); ok && x.Name == p {
								continue
							}
						}
					}
				}
				where := "first statement"
				if authTree {
					where = "first statement after the authorize call"
				}
				pass.Reportf(fd.Name.Pos(), "%s: %s must be `if err := %s.Valid(); err != nil { return nil, err }`", fd.Name.Name, where, p)
			}
		}
		return nil, nil
	},
}

var omitEmpty = &analysis.Analyzer{
	Name: "omitempty",
	Doc:  "handler Params/Result types return every field: no omitempty/omitzero in json tags",
	Run: func(pass *analysis.Pass) (any, error) {
		if !isHandlerPkg(pass) {
			return nil, nil
		}
		seen := map[*types.Named]bool{}
		var walk func(t types.Type)
		walk = func(t types.Type) {
			switch t := t.(type) {
			case *types.Pointer:
				walk(t.Elem())
			case *types.Slice:
				walk(t.Elem())
			case *types.Array:
				walk(t.Elem())
			case *types.Map:
				walk(t.Elem())
			case *types.Named:
				reportedInOwnPkg := t.Obj().Pkg() != pass.Pkg
				if reportedInOwnPkg || seen[t.Origin()] {
					return
				}
				seen[t.Origin()] = true
				walk(t.Underlying())
			case *types.Struct:
				for i := range t.NumFields() {
					opts := strings.SplitN(reflect.StructTag(t.Tag(i)).Get("json"), ",", 2)
					if len(opts) == 2 && (strings.Contains(opts[1], "omitempty") || strings.Contains(opts[1], "omitzero")) {
						pass.Reportf(t.Field(i).Pos(), "%s: drop omitempty/omitzero; API responses return every field", t.Field(i).Name())
					}
					walk(t.Field(i).Type())
				}
			}
		}
		for _, f := range pass.Files {
			if isTest(pass, f) {
				continue
			}
			for _, d := range f.Decls {
				if fd, ok := d.(*ast.FuncDecl); ok {
					if _, params, result, ok := handlerSig(pass, fd); ok {
						walk(params)
						walk(result)
					}
				}
			}
		}
		return nil, nil
	},
}

var slopComment = &analysis.Analyzer{
	Name: "slopcomment",
	Doc:  "no separator-banner comments and no commented-out code",
	Run: func(pass *analysis.Pass) (any, error) {
		for _, f := range pass.Files {
			if ast.IsGenerated(f) {
				continue
			}
			for _, g := range f.Comments {
				var lines []string
				for _, c := range g.List {
					text, ok := strings.CutPrefix(c.Text, "//")
					if !ok {
						break
					}
					if banner.MatchString(text) {
						pass.Reportf(c.Pos(), "drop the separator banner; split the file or let declarations speak")
					}
					lines = append(lines, text)
				}
				if len(lines) == len(g.List) && isCommentedCode(lines) {
					pass.Reportf(g.Pos(), "delete commented-out code; git keeps history")
				}
			}
		}
		return nil, nil
	},
}

// isCommentedCode reports whether a //-comment group of at least two lines parses as Go
// and has a code token: prose like "v2/otp/request" parses as division. Doc-comment code
// blocks (every line "//\t...") are examples, not dead code.
func isCommentedCode(lines []string) bool {
	if len(lines) < 2 || strings.HasPrefix(strings.TrimSpace(lines[0]), "Output:") {
		return false
	}
	example := true
	for _, l := range lines {
		if l != "" && !strings.HasPrefix(l, "\t") {
			example = false
		}
	}
	if example {
		return false
	}
	src := strings.Join(lines, "\n")
	if !codeToken.MatchString(src) {
		return false
	}
	fset := token.NewFileSet()
	if _, err := parser.ParseFile(fset, "", "package p\nfunc _() {\n"+src+"\n}", 0); err == nil {
		return true
	}
	_, err := parser.ParseFile(fset, "", "package p\n"+src, 0)
	return err == nil
}

var zeroDecl = &analysis.Analyzer{
	Name: "zerodecl",
	Doc:  "declare a zero-valued variable with var x T, not x := 0 / \"\" / false / T{}",
	Run: func(pass *analysis.Pass) (any, error) {
		check := func(stmts []ast.Stmt) {
			for _, s := range stmts {
				as, ok := s.(*ast.AssignStmt)
				if !ok || as.Tok != token.DEFINE || len(as.Lhs) != 1 || len(as.Rhs) != 1 {
					continue
				}
				id, ok := as.Lhs[0].(*ast.Ident)
				if !ok || id.Name == "_" || pass.TypesInfo.Defs[id] == nil || !isZeroLit(pass, as.Rhs[0]) {
					continue
				}
				typ := types.TypeString(pass.TypesInfo.Defs[id].Type(), func(p *types.Package) string {
					if p == pass.Pkg {
						return ""
					}
					return p.Name()
				})
				pass.Reportf(as.Pos(), "%s is the zero value; write var %s %s", id.Name, id.Name, typ)
			}
		}
		for _, f := range pass.Files {
			if ast.IsGenerated(f) {
				continue
			}
			ast.Inspect(f, func(n ast.Node) bool {
				switch n := n.(type) {
				case *ast.BlockStmt:
					check(n.List)
				case *ast.CaseClause:
					check(n.Body)
				case *ast.CommClause:
					check(n.Body)
				}
				return true
			})
		}
		return nil, nil
	},
}

// isZeroLit reports whether e spells a zero value literally: 0, "", false, T(0), or an
// empty struct literal. Named constants that happen to be zero (iota enums) don't count,
// and neither do []T{} / map[K]V{}, which are non-nil unlike their var form.
func isZeroLit(pass *analysis.Pass, e ast.Expr) bool {
	switch e := e.(type) {
	case *ast.BasicLit:
		v := pass.TypesInfo.Types[e].Value
		switch v.Kind() {
		case constant.String:
			return constant.StringVal(v) == ""
		case constant.Int, constant.Float:
			return constant.Sign(v) == 0
		}
	case *ast.Ident:
		return e.Name == "false" && pass.TypesInfo.Uses[e] == types.Universe.Lookup("false")
	case *ast.CallExpr:
		return len(e.Args) == 1 && pass.TypesInfo.Types[e.Fun].IsType() && isZeroLit(pass, e.Args[0])
	case *ast.CompositeLit:
		_, isStruct := pass.TypesInfo.TypeOf(e).Underlying().(*types.Struct)
		return len(e.Elts) == 0 && isStruct
	}
	return false
}
