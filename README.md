# gostyle

Go style checks for pgctx + arpc codebases, in one binary.

```sh
go get -tool github.com/xkamail/gostyle@latest

go tool gostyle lint ./...              # analyzers, non-zero exit on findings (-<rule>=false disables one)
go tool gostyle helpers ./...           # small, rarely-called funcs as JSON
go tool gostyle comments [-lines] ./... # narration-comment candidates as JSON (or file:line lines)
```

Rules: `pgiter`, `sqlstyle`, `discardederr`, `handlererr`, `stdlog`, `ctxbackground`,
`acronym`, `moneydecimal`, `validfirst`, `omitempty`, `slopcomment`, `zerodecl`.
Run `go tool gostyle lint help` for each rule's description.
