# gostyle

Go style checks for pgctx + arpc codebases, plus the AI-agent review skills that use them, in one binary.

```sh
go get -tool github.com/xkamail/gostyle@latest

go tool gostyle lint ./...              # analyzers, non-zero exit on findings (-<rule>=false disables one)
go tool gostyle helpers ./...           # small, rarely-called funcs as JSON
go tool gostyle comments [-lines] ./... # narration-comment candidates as JSON (or file:line lines)
go tool gostyle skills install          # write go-review / helper-review / comment-review into .claude/skills
```

Rules: `pgiter`, `sqlstyle`, `discardederr`, `handlererr`, `stdlog`, `ctxbackground`,
`acronym`, `moneydecimal`, `validfirst`, `omitempty`, `slopcomment`, `zerodecl`.
Run `go tool gostyle lint help` for each rule's description.

Per-repo settings are flags, e.g. `-moneydecimal.extra=credit,points` or `-validfirst.authfirst=admin`.

`skills install` overwrites only the files it ships. Keep repo-specific review rules in
`.claude/skills/go-review/references/local.md`; the go-review skill always reads it.
