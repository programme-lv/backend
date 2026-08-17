# Go style

Read these before changing Go in this repo:

1. [Effective Go](go/effective.md) — local copy of https://go.dev/doc/effective_go
2. [Go Doc Comments](go/comment.md) — local copy of https://go.dev/doc/comment
3. [Code review comments](go/review.md) — Go wiki review comments plus common extras

Live originals win if a local copy is stale.

## This repo

These override the copied guides when they conflict:

- Filenames: lowercase, no hyphens or underscores, one word when possible (`get.go`, `admin.go`, `types.go`).
- Do not prefix log or error strings with `failed to`.
- Wrap with `fmt.Errorf("open foo.txt: %w", err)`. Do not add `github.com/pkg/errors`.
- Do not add a Makefile for `fmt` / `vet` / `test` unless we decide to. Run `gofmt` and `go vet ./...` directly.
- Layout is `cmd/`, `modules/`, `common/`, `postgres/`. Do not restructure toward golang-standards/project-layout.
- `common` already exists. Do not add new `util`, `misc`, or `types` packages.
