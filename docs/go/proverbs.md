# Go Proverbs

Local copy of [Go Proverbs](https://go-proverbs.github.io/), from Rob Pike's [Gopherfest SV 2015 talk](https://youtu.be/PAAkCSZUG1c).

The live page wins if this list is stale. Where a proverb conflicts with [../go.md](../go.md) or [../errors.md](../errors.md), those win.

## Don't communicate by sharing memory, share memory by communicating.

Pass data on channels (or return values) instead of locking a shared buffer and hoping. Mutexes stay for protecting maps and similar state; see the next two.

## Concurrency is not parallelism.

`go` starts concurrency. Parallelism is a scheduler property. Do not add goroutines to “make it faster” without a measurement and a lifetime.

## Channels orchestrate; mutexes serialize.

Use a channel when one goroutine must wait for another or when you are distributing work. Use a mutex when many goroutines share a struct or map and only need exclusive access. The submission fan-out maps (`newSubmListeners`) are mutex + channel: mutex serializes the map, channels deliver.

## The bigger the interface, the weaker the abstraction.

Keep interfaces small at the point of use (`io.Reader`, `EvalRepo`). Module facades (`TaskService`, `SubmissionService`) are intentionally wide: they document the bounded context, not a reuse abstraction. Do not grow them with convenience methods another package could call through an existing method.

## Make the zero value useful.

Prefer types whose zero value is ready (`sync.Mutex`, `bytes.Buffer`). Constructors exist when the zero value cannot work (a nil `*pgxpool.Pool`).

## `interface{}` says nothing.

Name the type. `any` / `interface{}` is for JSON envelopes, `singleflight`, and `database/sql` argument lists — not for domain fields. `PaginatedResponse.Page` is `[]SubmListEntry`, not `interface{}`.

## Gofmt's style is no one's favorite, yet gofmt is everyone's favorite.

`gofmt` every Go file. Do not argue about wrapping.

## A little copying is better than a little dependency.

Do not add a package to avoid twenty lines. Copying a JSON shape is fine. Do not copy a second logging or error-code library; we already have `slog` and `srvcerror`.

## Syscall must always be guarded with build tags.

## Cgo must always be guarded with build tags.

## Cgo is not Go.

This repo does not use cgo or raw syscalls. Keep it that way unless there is no other option.

## With the unsafe package there are no guarantees.

Do not import `unsafe`.

## Clear is better than clever.

No `must*` helpers that panic. No `reflect` to skip an empty JSON body. No `defer` that infers commit vs rollback from a shadowed `err`.

## Reflection is never clear.

Do not use `reflect` to walk structs for JSON. Write `MarshalJSON` or let `encoding/json` see the fields. `httpfunc` is generics, not reflection.

## Errors are values.

Sentinels, `errors.Is`, `WithMsg`. See [../errors.md](../errors.md).

## Don't just check errors, handle them gracefully.

Log unexpected causes in the service layer, return a sanitized `srvcerror.E`. Do not `panic` after logging. Do not ignore `Commit` errors. HTTP logs only failures that originate there, including a failed write to the client.

## Design the architecture, name the components, document the details.

Modules, layers (`http` / `srvc` / `repo`), and filenames are the architecture. Package comments name the component. Details go in this `docs/` tree, not in a chat dump.

## Documentation is for users.

Godoc is for callers of the package. Operational truth lives in the sibling docs repo. Do not document the obvious.

## Don't panic.

`panic` is not an error-handling strategy. Process start uses `slog.Error` then `os.Exit(1)` (`Must*` in `conf`). Generated mocks may panic when a test forgot a return; that is their contract, not ours.
