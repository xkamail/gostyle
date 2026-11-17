---
name: helper-review
description: Use when the user says "/helper-review", "review helpers", "inline single-use helpers", or asks to find funcs that should be inlined, kept, or deleted. Runs `go tool gostyle helpers`, judges each candidate keep/inline/delete, applies the edits, verifies, and commits per package.
---

<!-- Installed by `go tool gostyle skills install`; local edits are overwritten on reinstall. -->

# Helper Review

`go tool gostyle helpers` finds the candidates; you make the call. The script is deliberately dumb — it lists every top-level func under 100 lines with ≤2 production call sites. Most candidates should be **kept**. Only change code where the rubric clearly says so.

## 1. Scan

```bash
go tool gostyle helpers ./...                                            # whole module
go tool gostyle helpers ./... | jq '[.[] | select(.pos | startswith("api/booking/"))]'  # one area
```

Always scan `./...` and filter: scanning a subtree hides callers in other packages, so exported funcs look dead.

Each entry: `func` (import path + name), `pos`, `lines`, `exported`, `callers` (production call sites), `testCallers`. Already excluded: methods, funcs used as values (callbacks, handler registration), generated and `_test.go` files, and funcs whose doc comment contains `// helperscan:keep`.

Work one package at a time. Read the func **and every caller** before deciding.

## 2. Decide

**Delete** — `callers` is empty **and** `testCallers` is 0. Confirm with `git grep` (reflection, `go:linkname`, build tags) first. Zero production callers but some test callers is not dead: it is a test hook (`SetPaymentClient`) or tested-only code — keep it, or ask the user if it looks abandoned.

**Inline** — all of these hold:
- One production caller (two only if both are tiny and identical in intent).
- The name says nothing the body doesn't: pass-through wrapper, renamed call, a few lines of plain sequencing.
- After inlining, the caller still reads top-to-bottom without losing its shape (rough guide: caller stays under ~80 lines, nesting doesn't deepen past 3).
- No test calls it directly (if one does, inlining means moving that test to the caller — only do it if the test gets simpler).

**Keep** — any of these:
- Names a domain concept or rule (`holdRoomAndPriceNights`, `verifySignupOTP`) — a reader skims the name instead of 50 lines.
- Owns a transaction, lock, retry, or `defer` scope that would leak into the caller.
- One branch of a dispatch (`forgotPasswordByEmail` / `forgotPasswordByPhone`) — siblings read better as a pair.
- Recursive, or called from a loop where inlining would bury the loop body.
- Exported from `pkg/` as a deliberate API, or from a handler package that another package imports.
- Inlining would duplicate a non-trivial body into 2 callers.

When unsure, keep. A wrong keep costs nothing; a wrong inline makes the caller harder to read.

Mark a deliberate keep only when it would otherwise be re-flagged and re-argued (it is borderline): add `// helperscan:keep <reason>` as the last line of the doc comment. Note the space after `//` — `//helperscan:keep` is a Go directive and is stripped from the doc text, so the scanner would miss it. Don't mark obvious keeps; the list is meant to be re-run.

## 3. Apply

- **Inline**: move the body into the caller, rename locals to fit, drop the func. Keep error wrapping and comments that explain *why*.
- **Delete**: remove the func and anything only it used.
- Remove imports your change made unused. Don't touch unrelated code.

## 4. Verify and commit (per package)

```bash
gofmt -w <changed files>
go build ./... && go tool gostyle lint ./...
go test ./<pkg>/...
```

Use the repo's own fmt/lint targets instead if it has them.

Commit each package separately: `refactor(<pkg>): inline single-use helpers` (or `remove dead helpers`). If tests fail, fix or revert that package before moving on.

## 5. Report

A table per package: func, decision, one-line reason. Then totals (kept / inlined / deleted) and anything you were unsure about.
