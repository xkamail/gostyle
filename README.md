# gostyle

Go style checks for pgctx + arpc codebases, plus the AI-agent review skills that use them, in one binary.

## Install

Per repo (recommended: it pins the version in `go.mod`, so lint and skills match everywhere, CI included):

```sh
go get -tool github.com/xkamail/gostyle@latest   # run from the repo root
go tool gostyle skills install                   # write the review skills into .claude/skills
```

Then point the repo's lint at it, e.g. a Makefile target:

```make
lint:
	go tool gostyle lint ./...
```

Globally, to run `gostyle` outside a repo that pins it:

```sh
go install github.com/xkamail/gostyle@latest
```

To upgrade a repo, re-run both per-repo commands with the new version (`@vX.Y.Z`), then lint and commit.
Check the installed version with `go tool gostyle version` (or `gostyle version` for the global binary).

## Usage

```sh
go tool gostyle lint ./...              # analyzers, non-zero exit on findings (-<rule>=false disables one)
go tool gostyle helpers ./...           # small, rarely-called funcs as JSON
go tool gostyle comments [-lines] ./... # narration-comment candidates as JSON (or file:line lines)
go tool gostyle skills install          # write go-review / helper-review / comment-review into .claude/skills
go tool gostyle version                 # print the pinned version (also stamped into each installed SKILL.md)
```

Rules: `pgiter`, `sqlstyle`, `sqlargs`, `discardederr`, `handlererr`, `stdlog`, `ctxbackground`,
`acronym`, `moneydecimal`, `validfirst`, `omitempty`, `slopcomment`, `zerodecl`, `exportfirst`.
Run `go tool gostyle lint help` for each rule's description.

Per-repo settings are flags, e.g. `-moneydecimal.extra=credit,points` or `-validfirst.authfirst=admin`.

## Skills

`skills install` writes three skills for AI agents. Claude Code reads `.claude/skills`; pass `-dir` for another agent's folder.

- `/go-review` reviews a diff, PR or package against the checklist, running `gostyle lint` first.
- `/helper-review` judges `gostyle helpers` candidates: keep, inline or delete.
- `/comment-review` judges `gostyle comments` candidates: keep, rewrite or delete.

`skills install` overwrites only the files it ships. Keep repo-specific review rules in
`.claude/skills/go-review/references/local.md`; the go-review skill always reads it, and install never touches it.
