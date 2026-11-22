---
name: go-review
description: Review Go code changes against personal Golang conventions (embedded checklist below). Use when user says "/go-review", "review my go code", "review go changes", or wants to audit a diff/PR against their Go rules.
---

<!-- Installed by `go tool gostyle skills install`; local edits are overwritten on reinstall. Put this repo's rules in references/local.md, which install never touches. -->

# /go-review

Review Go code changes against the user's personal Golang conventions.

## Rule Sources

The full checklist is embedded below — self-contained, no external rule files to load.

References live alongside this skill in `references/`. **None are auto-loaded except `local.md`.**

- `references/local.md` — **the repo's own rules. Always read it if it exists** and pass its absolute path to the subagent. It holds what this shared checklist can't know: the repo's file placement map, admin/authz surfaces, schema and task conventions, domain invariants, and any further repo references with the trigger for loading each. On conflict, `local.md` wins.
- `references/arpc-http-client.md` — load when the diff adds/changes a typed Go client that POSTs JSON to an arpc service and unwraps the `{ok, result, error}` envelope (look for a `Client` struct + `invoke`/per-method calls + `httpclient.Do` + envelope decoding).

## Flags

- (no flag) → **review only**, no edits. Default.
- `--fix` → after the review, apply 🔴 Must Fix items only. Safe edits with high confidence.
- `--fix-all` → apply 🔴 + 🟡 + 🟢 items. Use when you trust the rules and want everything aligned.

PR-scope review (`PR #123`) is **always review-only** — never `--fix` against a remote PR's working tree.

## What to Review

Default scope (in order of preference):
1. If user passes args like `PR #123` or a PR URL → review that PR via `gh pr diff <n>`
2. If staged changes exist (`git diff --cached --name-only` returns Go files) → review staged
3. Otherwise → review unstaged + staged (`git diff HEAD`) for `*.go` files
4. If nothing changed → review the current branch vs `main` (`git diff main...HEAD -- '*.go'`)

**Audit scope** — `audit <path|package>` (e.g. `/go-review audit api/booking` or `audit api/booking/refund.go`): review whole files at HEAD, no diff. Use for legacy code never covered by a diff review. The subagent Reads every non-test `.go` file under the path in full and runs the full checklist PLUS — mandatorily, not signal-gated — the Complexity Pass and Simplification Pass on every function that matches their activation signals. Start with the largest files (`wc -l`). For packages >3000 total lines, tell the user the file count and suggest narrowing before spawning. Audit scope is always review-only (`--fix` refused) — audit findings are refactor plans, not mechanical edits.

State which scope you picked in one line before diving in.

## Method

**Delegate to a Sonnet subagent.** Reading the full diff + every touched Go file is context-heavy work; it belongs in a fresh agent so the main session stays light.

1. Resolve scope first (in this session): run the `git`/`gh` command for the scope picked above and get the list of changed `*.go` files. If zero Go files → say so in one line and stop. Do NOT call the agent.
2. Parse flags. If `--fix` against a PR scope → refuse and explain (`--fix` only works on local changes).
2a. **Mechanical pass (in this session).** If `go tool gostyle version` succeeds, run the repo's gostyle lint: the Makefile `lint` target when it calls `gostyle lint` (it carries the repo's flags), else `go tool gostyle lint ./...`. Keep only findings in the changed files (audit scope: under the audited path). The tree being linted must be the code under review — for PR scope, skip this step unless `HEAD` is the PR's head commit, and say so in the scope line. If gostyle isn't installed or the lint fails to build, skip and say so in one line.
3. Spawn one `general-purpose` agent with `model: "sonnet"`. The prompt must be self-contained — the agent has no memory of this conversation. Include:
   - The absolute path to `references/local.md` (when it exists) with instruction to read and apply it, plus absolute paths to any **workload-specific references** whose trigger the changed files match (listed in Rule Sources or in `local.md`). Skip references that don't apply — don't make the agent read all of them.
   - The exact diff command to run, plus instruction to Read each changed file in full at HEAD for surrounding context.
   - The gostyle findings from step 2a (verbatim `file:line: message` lines, or "none"), with the instruction: report each one at the severity below, tagged `(gostyle <rule>)`; do NOT re-hunt the rules gostyle enforces — `pgiter`, `sqlstyle` (lowercase backtick SQL), `sqlargs` (query and Scan args one per line), `discardederr`, `handlererr`, `stdlog`, `ctxbackground`, `acronym`, `moneydecimal` (float money names), `validfirst`, `omitempty`, `slopcomment`, `zerodecl`, `exportfirst` (house rule, not Effective Go: exported funcs and vars above unexported), `testfile` (test file named after its source file). Spend the reading on what lint can't judge: placement, authz, transactions, races, money logic, file organization, tests. When step 2a was skipped, check those rules by reading as usual.
   - The full checklist (copy the sections below into the prompt — don't reference "the SKILL.md").
   - An explicit instruction to run the **Complexity Pass** section's auto-detection: scan every changed function for its activation signals (transactions, row locks, concurrency, async side effects, >100-line/4+-helper functions, money computed in 2+ places) and deep-review correctness on the ones that fire, in addition to the style checklist. This is the highest-value part of the review — the style checklist can pass a function that has a rollback-ordering or lost-update bug.
   - The output format and severity guide.
   - The "Don'ts" list.
   - **Fix mode** (only if `--fix` or `--fix-all` was passed): instructions in the "Fix Mode" section below.
4. Relay the agent's report to the user verbatim, prefixed with the scope line. If fix mode ran, also include the "applied / skipped" summary the agent produces.

## Fix Mode (when `--fix` or `--fix-all` is passed)

Add this block to the subagent prompt:

> After completing the review, apply edits with the Edit tool:
> - `--fix` → apply only 🔴 Must Fix items.
> - `--fix-all` → apply 🔴 + 🟡 + 🟢 items.
>
> Rules:
> - Apply only fixes you can make with a deterministic, mechanical edit (rename `Request`→`Params`, remove `omitempty`, lowercase SQL keywords, swap `&x`→`new(x)`, switch `http.DefaultClient.Do`→`httpclient.Do`, swap `errors.As(err, &t)`→`errors.AsType[*T](err)`, swap `json.Marshal`+`bytes.NewReader`→`json.NewEncoder(&buf).Encode`, etc.).
> - **Skip** fixes that require judgment or multi-file refactors (route renames that touch callers, type changes from `float64` to `decimal.Decimal`, handler signature rewrites that change call sites, schema redesigns). Mark them `SKIPPED — needs human review` and explain why in one line.
> - Never delete code beyond what the fix requires.
> - Never run `go test`, `go build`, or any formatter — pure surgical edits.
> - Do NOT touch files outside the changed-files list.
> - After edits, emit an "Applied / Skipped" summary:
>   ```
>   ✏️ Applied
>     - file.go:42 — removed `omitempty`
>   ⏭️ Skipped (needs human)
>     - file.go:88 — float64→decimal.Decimal: 6 call sites, manual decision
>   ```

## Checklist (from rules)

### Naming & Structure
- Handler types use `Params` / `Result`, not `Request` / `Response`
- Handler signature: `func XXX(ctx context.Context, p *XXXParams) (*XXXResult, error)`
- Route naming: `{module}.{action}` — action does NOT repeat the module (e.g. `booking.create`, not `booking.createBooking`)
- No `internal/` package — flat `api/` + `pkg/` only
- No `/api/v1` prefix in handler code (belongs in `mux.Group`)
- List item types named `Item`, not `BookingItem` / `RoomItem` etc.
- File section order, top to bottom: (1) sentinel error declarations (`var ErrXxx = ...`), (2) public function surface (exported funcs), (3) local/shared/reuse code (unexported funcs & helpers). Errors first, exported above unexported. Flag files where a sentinel error is declared mid-file or an unexported helper sits above an exported func.
- No alias imports unless there's a real package-name collision; collision case uses `{module}v1`, not `pkg{module}`

### File Placement (minimal — the repo's full map is in `references/local.md`)
- `api/{module}/*.go` holds the arpc handler funcs. Routes are registered centrally (e.g. `api/handler.go`'s `Mount(mux, am)`); `cmd/api/main.go` creates the mux + `arpc.Manager` and only calls `api.Mount(...)`.
- A route whose handler reads the caller's identity from ctx (e.g. `auth.UserID(ctx)`) MUST register on the authed group (`a := mux.Group("", am.Middleware(authMiddleware))` → `a.Handle(...)`), not the public `mux.Handle(...)` or an optional-auth group where the identity may be absent. Wrong group = empty/panicking auth context at runtime → 🔴. Per-route middleware chains as `.Middleware(mw).Handle(...)`; a shared prefix+middleware as `mux.Group(prefix, mw...)`.
- Logic reused by both a handler and a worker task lives in `pkg/{module}/` — duplicating it in the handler is a 🔴.
- Every admin-ish surface (staff, back office, partner API) delegates to `pkg/` — reimplementing domain logic in a handler package is a 🔴.

### File Organization (one domain per file)
- A package file should hold ONE domain. When a single file accumulates many handlers spanning distinct domains, split it into per-domain `*.go` files (e.g. `booking.go`, `availability.go`, `payment.go`, `refund.go`, `invoice.go`) named for the domain — not one catch-all `<package>.go` holding everything.
- Mirror the existing `*_test.go` layout: if `book_test.go` / `get_room_test.go` exist, the source belongs in matching domain files. Source-file names should track test-file domains.
- The package's core file (`<package>.go`) keeps only cross-domain shared declarations: sentinel errors, shared types, and helpers used by 2+ domains. A helper used by exactly one domain lives in that domain's file (and at the bottom, per file section order).
- Group related handlers of the SAME domain together in one file (e.g. `Book`/`Cancel`/`Reschedule`/`Bookings`/`MyBooking` → `booking.go`) rather than one file per handler.
- Flag a 🟡 when one non-test file holds **3+ exported handlers of distinct domains** OR exceeds **~600 lines** while mixing domains — and name the concrete split (list the domain files). A long file that is genuinely one cohesive domain is fine; size alone is not a violation.
- Escalate to 🔴 when a mixed-domain file passes **~1200 lines** or **~1500 lines with 3+ domains** — at that size the file is a maintenance and merge-conflict hazard, not a style nit. Do NOT soften this into "would not do it opportunistically"; state the concrete split as a required follow-up (e.g. `booking.go` → `booking.go` shared types + `availability.go` + `payment.go` + `refund.go` + `invoice.go`, with shared sentinel errors/types left in the core file). The reviewer's job is to name the split, not to defer it.

### JSON & Types
- No `omitempty` in JSON struct tags — every field always emitted
- `shopspring/decimal` for money — prices, fees, balances, payouts — never `float64` or raw `int*` for these. `references/local.md` may list more money-like fields or types to leave alone
- `int*` only for IDs, indexes, levels, enum counters, pagination

### Handlers & Validation
- `if err := p.Valid(); err != nil { return nil, err }` is the first statement in every handler, unless `references/local.md` names a surface that authorizes first
- Authorization comes right after `Valid()`, using the repo's permission check (see `references/local.md`). A mutating staff/admin handler with no authz call → 🔴
- Errors go through arpc codes — no `fmt.Errorf` for user-facing errors; return `err` directly or use `arpc.NewErrorCode`
- No empty-string guards on internal args (`if x == "" { return err }`) — trust callers; validate only at HTTP boundary (`Valid()`)
- The caller identity from ctx (e.g. `auth.UserID(ctx)`) is never checked for empty on authed-group routes; middleware guarantees it. On an optional-auth group it can be empty, and that check is legitimate
- Prefer `var r XxxResult` + field assignment over struct literal init for multi-line population

### Validation (`Valid()` method)
- Use `github.com/moonrhythm/validator`: `v := validator.New()`, one or more `v.Must(cond, "message")` calls (or `v.Mustf(cond, "message %d", i)` for messages with loop indices), then `return v.Error()`
- Field normalization (`strings.TrimSpace`, `strings.ToUpper`, etc.) happens inline at the top of `Valid()`, mutating `p`'s fields directly, before the `v.Must` checks that depend on it
- Nested/composed param structs (e.g. `p.Sort`, `p.Paginate`) delegate via `if err := p.Sort.Valid(); err != nil { return err }` rather than re-checking their fields inline
- ID fields are validated by format (UUID, fixed length — match the package's local helper), not just `!= ""`

### SQL & Database
- All SQL keywords lowercase (`select`, `from`, `where`, `join`)
- No `::text` casts — Go's `database/sql` handles conversion
- Dynamic queries (optional filters, search, conditional WHERE) → `pgstmt`. Raw SQL via `pgctx` only for fixed queries
- One column per line in SQL strings; after a multi-line SQL string, each query arg on its own line and `)` on its own line (`` `,\n\tid,\n).Scan( ``); one arg per line in `Scan(...)`, even a single one after a multi-line SQL string
- Multi-row reads use `pgctx.Iter` — never `pgctx.Query` + manual `rows.Next()`
- Single row → `pgctx.QueryRow`. Writes → `pgctx.Exec`
- Slices initialized empty before `pgctx.Iter`: `items := make([]Item, 0)` (not `var items []Item`)
- `pgsql.IsUniqueViolation(err)` checked BEFORE generic error return; returns `ErrDuplicate`
- Schema migrations are idempotent (`if not exists` everywhere); the repo's file layout and migration rules are in `references/local.md`
- JSON/JSONB columns use `pgsql.JSON(&field)` in both Scan and query args — never manual `json.Marshal`/`Unmarshal`
- Nullable scans use `pgsql.Null(&field)`

### pgstmt Patterns
- `b.Columns("col1", "col2")` — separate args, not one comma-joined string
- Select: `pgstmt.Select(func(b pgstmt.SelectStatement) { ... }).IterWith(ctx, func(scan pgsql.Scanner) error { ... })`
- Update: `.ExecWith(ctx)`
- `b.Where(func(c pgstmt.Cond) { c.Eq("col", val) })`; multiple conds → `c.Mode().And()` / `c.Mode().Or()`
- Conditional filters guarded with `if` inside the Where callback
- Null checks → `c.IsNull("col")` / `c.IsNotNull("col")`, never `c.Raw("col is null")`
- Search uses a separate Where block with `c.Mode().Or()` + `c.ILike(...)`
- Join → `b.Join("table alias").On(func(c pgstmt.Cond) { c.EqRaw("a.col", "b.col") })`
- Partial update: `if p.Field != nil { b.Set("field").To(*p.Field) }`
- Raw SQL values: `b.Set("updated_at").ToRaw("now()")`

### Pagination (`pkg/rest`)
- Params embed `*rest.Paginate` with json tag `"paginate"`; `Valid()` checks `p.Paginate != nil`
- Result embeds `*rest.Paginate` with json tag `"paginate"`; initialized as `r.Paginate = p.Paginate`
- `r.Items = make([]Item, 0)`
- Count query first via `r.Paginate.Count(func() (int64, error) { ... })` using `pgstmt.Select` with `count(*)`
- Data query applies `r.Paginate.Do(b)` inside the `pgstmt.Select`

### General Patterns
- Inline over single-use helpers — extract only with 2+ real call sites
- Pointers via `new(value)`, not `&variable` (e.g. `pgsql.Null(new(adminID))`)
- Prefer `slices` package over manual for-loops
- Prefer `json.NewDecoder(r.Body).Decode(&v)` over `io.ReadAll` + `json.Unmarshal` when reading a body
- Prefer `json.NewEncoder(&buf).Encode(v)` into a `bytes.Buffer` over `json.Marshal` + `bytes.NewReader` when building a request body
- Type-assert errors with `errors.AsType[*T](err)` (Go 1.26), not `errors.As(err, &target)`
- Env vars via `configfile.NewEnvReader()` — never `os.Getenv`
- HTTP via `httpclient.Do` — never `http.DefaultClient.Do`
- Routing via `github.com/moonrhythm/httpmux` (`httpmux.New()`, `.Group(prefix, mw...)`, `.Middleware(mw).Handle(...)`) — never stdlib `http.ServeMux` / `http.NewServeMux`. Middlewares are `httpmux.Middleware` (`func(http.Handler) http.Handler`) applied via `.Middleware(...)` / `.Group(...)`, not hand-wrapped `http.Handler`s. Method routing uses the `"POST /module.action"` pattern string
- Logging via `slog.InfoContext` / `slog.ErrorContext` — never the non-context variants
- Shared logic between worker and API in `pkg/{name}/`
- REST utilities (`Paginate`, `Sort`, etc.) in `pkg/rest`

### Effective Go idioms

Baseline from [Effective Go](https://go.dev/doc/effective_go) + [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments). The house rules above win on any conflict (e.g. `var r XxxResult` over literals, `new(value)`). Flag only what the diff introduces.

**Names**
- Package names: short, lowercase, one word, no `_`/mixedCaps, no `util`/`common`/`base`. No stutter — `booking.Service`, not `booking.BookingService`; `chat.New`, not `chat.NewChat` → 🟢.
- MixedCaps only, never `snake_case`. Initialisms keep one case: `ID`, `URL`, `HTTP`, `JSON`, `API` (`userID`, `apiURL`, `ServeHTTP` — not `userId`, `ApiUrl`) → 🟢.
- Getters drop `Get`: method `Owner()`, setter `SetOwner()`. (Arpc handler funcs named `Get`/`List` are route actions, not getters — don't flag.)
- One-method interfaces take the `-er` name (`Reader`, `Notifier`). Don't reuse a canonical method name (`String`, `Read`, `Write`, `Close`, `Error`) with a different signature or meaning → 🟡. Short names for short scopes (`i`, `r`, `p`); descriptive names for wide scopes and package-level vars.
- Receiver name: 1–2 letters from the type, same across all methods; never `this`/`self`.

**Control flow**
- Happy path at the left margin: handle the error and `return` early; no `else` after a block that ends in `return`/`break`/`continue` → 🟢 (mechanical under `--fix-all`).
- Scope short-lived values to the statement: `if err := f(); err != nil`, `if v, ok := m[k]; ok`.
- `:=` in an inner block that re-declares an outer `err`/result var shadows it — the outer value is never set → 🔴 when the outer one is returned or checked later.
- A 3+ arm `if/else if` chain on one value → expression-less `switch` or `switch v`. Type dispatch → type switch, not repeated assertions.
- Assertions and map reads that can miss use comma-ok (`v, ok := x.(T)`); a bare `x.(T)` that can panic on untrusted input → 🔴.

**Functions & defer**
- `defer` args are evaluated when the `defer` runs, not at return (`defer log(time.Since(start))` logs ~0) — wrap in a closure when you want the late value → 🟡. Deferred calls run LIFO.
- `defer` cleanup right after the acquire succeeds (`f, err := os.Open(...); if err != nil {...}; defer f.Close()`). `defer` inside a loop holds every resource until the function returns → 🟡 (move the body into a func or close explicitly).
- Named results only when they document same-typed returns or a deferred func must modify them; no bare `return` in functions longer than a few lines → 🟢.
- Don't pass a pointer to save copying a small struct/string/slice/map; pass a pointer when the callee mutates or the struct is large.

**Data**
- Zero-valued locals use `var`: `var ordinal int`, `var reason string`, `var res XxxResult` — not `ordinal := 0` / `reason := ""` / `res := XxxResult{}`. `:=` is for a real initial value (`name := "aa"`). Empty slices/maps keep `[]T{}` when non-nil matters. Enforced by gostyle `zerodecl` → 🟢.
- Known final length → preallocate: `make([]T, 0, len(src))` (still satisfies the "empty, not nil" rule for results).
- `append` may reallocate — always assign the result back (`s = append(s, x)`); appending to a sub-slice that shares a backing array with a live slice silently overwrites it → 🔴.
- Map/type-assertion misses return the zero value; use comma-ok when zero is a valid value (`qty, ok := m[id]`).
- Receivers: pointer if any method mutates or the struct is large/holds a mutex; then **all** methods use pointer receivers — mixed receiver kinds on one type → 🟢. Never copy a struct containing `sync.Mutex`/`sync.WaitGroup` (value receiver, range copy, pass by value) → 🔴.
- Zero value should be useful: prefer structs that work without a constructor; don't add `NewX()` that only returns `&X{}`.
- `iota` enums get a `String()` via `stringer` (`//go:generate stringer`, `make generate`), not a hand-written switch.
- A `String()` method that formats its own receiver with `%v`/`%s` (`fmt.Sprintf("%v", t)`) recurses forever → 🔴; convert first (`string(t)`, or a field).
- Struct embedding promotes methods, but the receiver is still the inner type — flag embedding used as "inheritance" where the outer type expects its own fields to be seen → 🟡. Embed to expose the inner API; use a named field when you only need to delegate a method or two.

**Interfaces**
- Define interfaces where they're consumed, not next to the implementation; keep them small. Functions accept interfaces and return concrete types — returning the interface is right only when the type has no behavior beyond it (Effective Go "Generality", e.g. swappable gateway/SMS providers) → 🟡 otherwise.
- No interface with a single implementation "for mocking" unless a test fake actually implements it (compile-time `var _ I = T{}` check — only needed where no static conversion already proves it).

**Errors**
- Error strings: lowercase, no trailing punctuation, and name their origin (`"booking: unknown status"`, `fmt.Errorf("load room: %w", err)`) — not `"failed to …"`/`"error: …"` noise → 🟢.
- Internal wrapping uses `%w` (never `%v` for an error you might `errors.Is` later); compare with `errors.Is`, never `==` or `err.Error()` string matching → 🟡. (User-facing errors are still `arpc` codes — see Handlers.)
- Handle an error once: log **or** return, not both → 🟡.
- `panic` only for programmer errors / impossible states at init (`regexp.MustCompile`, missing required config at startup). A `panic` on a runtime/input path → 🔴. `recover` only at a goroutine/request boundary, inside a deferred func.
- A panic in a goroutine spawned from a handler/task kills the whole process — the httpmux/asynq recovery only covers the request goroutine. Long-lived or fire-and-forget `go func` needs `defer func() { if r := recover(); r != nil { slog.ErrorContext(...) } }()` → 🟡.

**Concurrency** (details in the Complexity Pass)
- "Do not communicate by sharing memory; instead, share memory by communicating": prefer handing ownership through a channel or `errgroup` over a mutex-guarded shared slice when it's the simpler shape.
- Every goroutine has a clear exit — ctx cancellation or a closed channel. A goroutine blocked forever on a send nobody reads (unbuffered chan, early-returning receiver) is a leak → 🔴. Only the sender closes a channel.
- Unbounded `go` per item (one goroutine per row/request) → bound it: `errgroup.SetLimit(n)`, a buffered-channel semaphore, or a fixed worker pool reading one channel → 🟡.
- `context.Context` is the first param named `ctx`; never stored in a struct field; never `nil` (use `context.Background()`/`TODO()`).

**Misc**
- `gofmt`/`goimports -local <module path>` clean (import groups: stdlib, third-party, local).
- Blank imports (`_ "pkg"`) only in `main` or tests, with a one-line reason.
- Exported names in `pkg/` carry a doc comment starting with the name (`// Charge debits …`) → 🟢 when a new exported func/type lacks one.

### Errors are never silently ignored
- Every returned `error` is checked or explicitly assigned to `_` with a reason in a comment

### Complexity Pass (correctness — auto-detected)

The rest of this checklist is convention/style matching. This section is different: it reviews **logic correctness**, and it only activates when the diff carries complexity signals. Run it **in addition to** the style checklist — never instead of it.

**Activation.** Turn this pass on for any changed function that matches ONE OR MORE of:
- a DB transaction — `pgctx.RunInTx` / `RunInTxOptions` / `MaxAttempts` / `TxOptions`
- row locking — `for update`, `lockXxx` helpers, `select ... for update`
- concurrency — `go func`, `sync.WaitGroup`, channels, `errgroup`, shared mutable state across goroutines
- async side effects — `task.Enqueue*` / `task.Enqueue`, asynq enqueue, realtime/SSE publish, notification, any fire-and-forget started from a handler
- a function **> ~100 lines** or that fans out to **4+ same-package helpers**
- money/quantity arithmetic that is **computed in more than one place** for the same booking/invoice/ledger (totals, discounts, refunds, balances)

If none match, skip this section silently (don't mention it). If any match, name which signal fired for each function you deep-review.

**1. Transaction correctness**
- Side effects that must not fire on rollback (enqueue, external call, cache write, notification) run **after** the `RunInTx` closure returns `nil` — never inside the closure. Pattern: capture what you need in a struct + a `runX bool` inside the tx, dispatch after. Flag any enqueue/HTTP/dispatch **inside** the closure → 🔴 (fires even when the tx aborts).
- `MaxAttempts: 1` under a serializable/repeatable-read tx means a concurrent-update serialization error (`pq` `40001` "could not serialize access") surfaces **raw** to the caller. Confirm the caller/handler maps it to a user-facing `arpc` code (use `pgsql.IsSerializationFailure(err)` — don't hand-unwrap `*pq.Error`; the package also has `IsErrorCode`/`IsErrorClass` for other codes) or the test asserts the abort is acceptable — a raw serialize error reaching the client is 🔴; an intentional single-attempt with a documented "loser is rejected" contract is fine (note it).
- Lock ordering: when a tx `FOR UPDATE`-locks **two or more tables** (e.g. `bookings` then `rooms`), every code path that locks the same pair must lock them in the **same order**, or two concurrent txs deadlock. Flag inconsistent ordering → 🔴.
- The rows locked `FOR UPDATE` must be the rows later mutated. Locking table A but `update`-ing table B under contention is a lost-update bug → 🔴.
- `pgctx.RunInTx` (and `RunInTxOptions` without an explicit isolation) is **Serializable** with a few immediate retries. Under real concurrency on the same rows those retries can run out with `40001`. Whenever that error is swallowed — logged and `continue`d, or a task handler returns `nil` — the write is silently lost → 🔴. Either pick an isolation/locking scheme that waits instead of aborting, or return the error so the caller retries (and make that retry idempotent).
- `select ... for update` locks nothing when the row doesn't exist yet, so "read-or-default, modify, upsert" lets concurrent first-time callers all start from the default → lost update 🔴. Make the row exist first (`insert ... on conflict do nothing`), then `select ... for update`, then `update` — under `sql.LevelReadCommitted`, so a waiter re-reads the committed value once the lock is released. Back it with a test that fires concurrent callers and asserts exact totals.

**2. Concurrency & races**
- Distinguish **advisory** checks from **authoritative** ones. An availability/capacity/balance check (room capacity, seat count, account balance) that does NOT take a lock or write is advisory — it can pass for N concurrent callers. The authoritative guard must be a locked read + conditional write **inside** the same tx that commits the effect. Flag when the only guard against overbooking/overspend is an unlocked `select` → 🔴.
- TOCTOU: any "check then act" on shared state (room capacity, check-in, booking status, account balance, uniqueness) where the check and the act are not in one atomic locked/tx step, or not backed by a DB constraint → 🔴.
- Goroutines: every `go func` writing shared state needs a mutex/channel; every spawned goroutine's error must be collected (channel / errgroup), not dropped. `WaitGroup.Add` before the `go`, `Done` via `defer`. Flag races and dropped goroutine errors → 🔴.

**3. Logic consistency**
- A value computed **twice** for the same entity (classic: a price estimated in the handler, then recomputed authoritatively under lock inside the transaction) must reconcile — the authoritative one wins and the estimate must not leak into the persisted row. Flag when the two can silently diverge (different inputs, different rounding, one path skips a cap) → 🟡, or 🔴 if the wrong one is persisted/charged.
- Money rounding (prices, fees, refunds, payouts): rounding direction and precision consistent across the flow; a refund never exceeds what was paid; a cap is actually applied, not just computed. `decimal` compared with `.Equal`/`.GreaterThan`, never `==`.
- Splits (a payment across line items, guests across rooms) must sum back to the whole — flag off-by-one / leftover-unit bugs. Domain ordering rules (tie-breaks, priorities) follow the repo docs named in `references/local.md`. Status/enum: every branch assigns a reachable terminal status; no path leaves status unset.

**4. Failure & idempotency**
- Partial failure: if step 3 of 5 fails, do steps 1–2 leave dangling rows / held slots / debited balance? Effects that must be atomic belong in ONE tx; effects that can't (external calls) need a compensating path or an idempotent retry.
- Retried async tasks (asynq redelivers) must be idempotent — a task that increments a counter, applies a status transition, sends a payout, or credits a balance without an idempotency key/guard double-applies on retry → 🔴.
- `pgsql.IsUniqueViolation` (and other expected DB errors) mapped to a domain error **before** the generic `return err`, so a race that trips a unique constraint becomes a clean `ErrDuplicate`, not a 500.
- User-facing errors are `arpc` codes; a raw `pq`/driver/`fmt.Errorf` error reaching the client leaks internals → 🔴.

### Simplification Pass (large files / large functions)

When the Complexity Pass activates on a function >~100 lines, or any changed file is a mixed-domain file >~600 lines, also hunt for **generated bloat** — code that looks necessary but does nothing. These checks found ~100 dead/duplicated lines and 3 real bugs in a single 1885-line audit; they pay for themselves.

- **Write-only param fields.** For every field of an internal params struct (like an internal `xxxParams`), verify it is *read* inside the callee — not just populated at the call site. A write-only field is dead, and so is the entire upstream computation chain that feeds it (trace it: the dead field often keeps a whole helper + adjustment blocks alive in 2+ callers). Flag the field AND the chain → 🟡, with exact lines to delete.
- **Estimate-then-recompute.** When a handler computes a value pre-transaction and the transaction recomputes it authoritatively (post-lock availability, in-tx price), the pre-tx result is usually dead except for early error surfacing and cheap flags. Keep the call only for what's actually consumed; delete the arithmetic that feeds nothing.
- **Zero-caller functions.** grep every unexported function in the touched file for callers; zero call sites = delete → 🟡.
- **Sibling-handler duplication.** When two handlers share the same flow tail (e.g. guest-side and staff-side booking changes sharing validate booking → validate room → apply change → enqueue notification), quantify the verbatim overlap. ≥2 real call sites justifies ONE shared flow function (the "inline over single-use helpers" rule cuts the other way here). Two drifting copies of an apply path is a correctness time bomb, not a style choice.
- **Refactor gate: pins first.** Never propose a behavior-touching simplification without stating which existing test pins it (file:test name) or naming the missing test that must land FIRST. Untested branches to check for specifically: caps that every fixture sets to zero (fixtures that never exercise the limit), time-window guards, reject paths where only the accept path is tested, and boundary values (fully booked, exact capacity, zero balance).
- **Money invariants.** Balances, totals and payouts must be provably ≥ 0 and conserve what was paid in; a missing clamp is 🔴.
- **`for update` needs deterministic order.** `where id = any($1) ... for update` without `order by id` deadlocks two overlapping multi-row updates. A comment claiming "consistent order" doesn't make it true — the SQL must. 🔴.

**Generation-time corollary** (applies when this skill's owner is *writing* a complex endpoint, not just reviewing): plan the domain split BEFORE writing (orchestrator + per-domain files); give every computed value exactly one authoritative computation site; when a second handler needs the same flow, extract the shared function immediately instead of pasting; and never populate a param field "for later" — add fields when a reader exists.

**How to report.** Complexity findings are correctness findings — severities follow the guide (a real bug is 🔴 even if the style checklist is spotless). For each, give the failure scenario in one line (concrete inputs → wrong outcome), not just the rule. If you deep-reviewed a complex function and found it correct, say so in one ✅ line naming what you verified (e.g. "createOrder: side effect correctly dispatched post-commit; lock order consistent") — this is the one place restating a passing check is worth it, because it tells the user the hard path was actually examined.

### Test Conventions (`*_test.go`)
- One test file per source file: `foo.go` → `foo_test.go`; white-box tests of unexported code → `foo_internal_test.go`; `main_test.go` only for `TestMain`. A new scenario, bug or feature (rollback, race, authz, coverage) is a `t.Run` subtest in `foo_test.go`, not a new `foo_<scenario>_test.go`, and a test file is never named for a handler or func (`get_match_test.go`) — it goes in the test file of the source file that declares it
- External test package: `package xxx_test`
- `tu.Setup()` + `defer tc.Teardown()` at the top of every `TestXxx`
- `t.Parallel()` as the first call inside `TestXxx`
- Subtests via `t.Run("case name", func(t *testing.T) { ... })`; standard names: `"success"`, `"ok"`, `"not found"`, `"validation"`, `"duplicate"`, `"expired"`
- `require` for fatal checks (setup, seed `NoError`); `assert` for value checks
- Sentinel errors via `assert.ErrorIs(t, err, pkg.ErrXxx)`
- Unordered slices → `assert.ElementsMatch`; time → `assert.WithinDuration`
- `decimal.Decimal` comparison: `assert.True(t, expected.Equal(got), ...)` — not `assert.Equal`
- Seed via raw SQL `pgctx.Exec` with `on conflict ... do nothing`
- Fixtures seeded in the shared test-util package (`pkg/tu/`: `tu.go` setup + per-domain helpers like `tu/user.go`) — not inline in individual test files
- Fakes for external services in `pkg/tu/` with `var _ Interface = fakeImpl{}` compile-time check; unused methods `panic("implement me")`
- Redis isolated via a unique key prefix per test, set up in `tu.Setup()`; teardown deletes keys by that prefix
- Payment gateways / HTTP clients: mock structs defined in the test file using a functional-options pattern (`func setTestXxx(t *testing.T, ctx context.Context, opts ...func(*mockXxx)) context.Context`)
- Mocks injected via context setters (`payment.SetClient`)
- Concurrency tests use `sync.WaitGroup` + buffered channels: spawn goroutines, collect results/errors in channels, `wg.Wait()`, then assert both success count and error count
- Auth via the auth package's context setter (e.g. `auth.WithUserID(ctx, id)`)
- Test helpers at bottom of file, marked `t.Helper()`
- No `t.Skip` — tests must fail loudly when infra is missing

## Output Format

```
Scope: <staged | unstaged | PR #N | branch vs main> — <N Go files> — gostyle lint: <N findings | skipped: reason>

🔴 Must Fix
  - file.go:42 — <rule violated>. <one-line fix>.

🟡 Should Fix
  - file.go:88 — <rule violated>. <one-line fix>.

🟢 Nits
  - file.go:120 — <minor convention drift>.

✅ Looks Good
  - <one-line note on what was done well, only if there's something genuinely notable>
```

Severity guide:
- gostyle findings: `discardederr`, `handlererr`, `ctxbackground`, `validfirst`, `moneydecimal` → 🔴; `pgiter`, `sqlstyle`, `omitempty`, `stdlog` → 🟡; `acronym`, `zerodecl`, `slopcomment`, `exportfirst` → 🟢
- 🔴 **Must Fix** — bugs, security, incorrect SQL, wrong decimal usage, missing `Valid()`, ignored errors, **Complexity-Pass correctness defects** (side effect inside tx, lost update, oversell via advisory-only check, TOCTOU, non-idempotent retried task, raw driver error to client, mixed-domain file past ~1200 lines)
- 🟡 **Should Fix** — naming, omitempty present, pgstmt patterns, test conventions, divergent-but-not-yet-charged duplicate computations, mixed-domain file 600–1200 lines
- 🟢 **Nit** — style drift, ordering, helper extraction calls

If no Go files changed, say so in one line and stop. Don't invent findings.

## Don'ts

- Don't auto-fix **unless** `--fix` or `--fix-all` was explicitly passed.
- Don't restate rules the code already follows. Silence = passing.
- Don't review non-Go files even if they're in the diff.
- Don't run `go test` or `go build` — apart from the gostyle lint step, this is a static review.
- Don't apply `--fix` against PR scope — local edits only.
