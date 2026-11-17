---
name: comment-review
description: Use when the user says "/comment-review", "review comments", "remove AI slop comments", or asks to find narration comments to delete or rewrite. Runs `go tool gostyle comments`, judges each candidate keep/rewrite/delete, applies the edits, verifies, and commits per package.
---

<!-- Installed by `go tool gostyle skills install`; local edits are overwritten on reinstall. -->

# Comment Review

`go tool gostyle comments` finds the candidates; you make the call. Rule: a comment earns its place only by saying something the code can't — a non-obvious why, a gotcha, a constraint from outside the code. Narration of what the code plainly does goes.

The hard cases (separator banners, multi-line commented-out code) are already a CI lint — `gostyle lint`'s `slopcomment` rule. This skill handles what needs judgment.

## 1. Scan

```bash
go tool gostyle comments ./...                                                   # whole module
go tool gostyle comments ./... | jq '[.[] | select(.pos | startswith("api/booking/"))]'  # one area
```

Each entry: `pos`, `reason`, `text` (the comment), `code` (the line it describes: the code before it for a trailing comment, else the next code line). Reasons:

- `narration` — opens with an action verb (`check`, `create`, `loop`, `return`, …) and has no why-marker.
- `restates-code` — most of its words are identifiers in `code`.
- `restates-name` — an unexported func's doc comment that opens with the func name.
- `commented-out-code` — a single line that parses as Go.

Already skipped: exported doc comments and struct field comments (godoc and doc generators read them), `_test.go`, generated files, comments containing a why-marker (`because`, `must`, `never`, `:`, `—`, …).

Work one package at a time. Read the surrounding func before deciding — the scanner sees one line of code, you see the whole block.

## 2. Decide

**Delete** — any of these:
- It says what the next line says (`// check availability` over `rooms.Available(...)`, `// create user` over `upsertUser(...)`).
- A section label over a block whose calls already name the section (`// user` above `user.*` routes).
- Commented-out code. Git has history.
- A `restates-name` doc that is the name in sentence form (`// generateToken for given user id`).

**Rewrite** — the comment points at something real but buries it in narration. Keep only the why: `// Validate redirect URL for security. Allow HTTP in development` → `// http:// is allowed only outside production: local OAuth callbacks.` A rewritten comment normally contains a why-marker, so the next scan won't flag it again.

**Keep** — any of these:
- It explains a constraint the code can't show: an external API quirk, a protocol rule, a units/formula note (`// 60 req/min => 1 req/s sustained`), a security reason.
- A `restates-name` doc on a func whose name is opaque but whose contract matters to callers (what nil means, what it locks, what it returns on not-found) — consider renaming instead.
- Deleting it would make a reader re-derive something non-obvious.

When unsure, keep. A wrong keep costs a line; a wrong delete loses a reason someone needed.

If a narration comment papers over unclear code (a long condition, an unnamed tuple, an opaque regex), prefer fixing the code — a named predicate, named results, a test pinning the case — then delete the comment. Only do this inside the package you're on.

## 3. Verify and commit (per package)

```bash
gofmt -w <changed files>
go build ./... && go tool gostyle lint ./...
go test ./<pkg>/...
```

Use the repo's own fmt/lint targets instead if it has them.

Commit each package separately: `refactor(<pkg>): drop narration comments`. Comment-only diffs can't break tests, but code you renamed or extracted can.
