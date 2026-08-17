# Go style

Read these before changing Go in this repo:

1. [Effective Go](go/effective.md) — local copy of https://go.dev/doc/effective_go
2. [Go Doc Comments](go/comment.md) — local copy of https://go.dev/doc/comment
3. [Code review comments](go/review.md) — local copy of https://go.dev/wiki/CodeReviewComments
4. [Go Proverbs](go/proverbs.md) — local copy of https://go-proverbs.github.io/

Live originals win if a local copy is stale.

## This repo

These override the copied guides when they conflict:

- Filenames: lowercase, no hyphens or underscores, one word when possible (`get.go`, `admin.go`, `types.go`).
- Do not prefix log or error strings with `failed to`.
- Wrap with `fmt.Errorf("open foo.txt: %w", err)` **inside** repo and other non-HTTP Go. Do not wrap a `srvcerror.E` in `%w` before returning it to HTTP. Do not add `github.com/pkg/errors`.
- Layer rules for client errors: [errors.md](errors.md). Log levels and what not to log at Info: [logging.md](logging.md).
- Do not add a Makefile for `fmt` / `vet` / `test` unless we decide to. Run `gofmt` and `go vet ./...` directly.
- Layout is `cmd/`, `modules/`, `common/`, `postgres/`. Do not restructure toward golang-standards/project-layout.
- `common` already exists. Do not add new `util`, `misc`, or `types` packages.
