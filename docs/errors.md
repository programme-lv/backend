# Errors by layer

Flow: repo returns a Go `error` → service logs the real cause if it is unexpected → service returns a **sanitized** `srvcerror.E` → HTTP (or another service) forwards that to the client.

Repo does not log the internal-server-error reason.
Service does.
HTTP logs only failures that originate in HTTP (bad JSON, auth, write to the client).
It does not log the Postgres or object-store cause; that log line already happened in `srvc`.

Do not prefix logs or error strings with `failed to`.
Do not leak SQL, `pgx`, or `fmt.Errorf` strings to JSON.

Package [`common/srvcerror`](../common/srvcerror/srvcerror.go) is the sanitized error.
It carries a snake_case `code`, a Latvian `message`, and an HTTP status.
`errors.Is` matches on **code**, not identity or message.
`WithMsg` copies the sentinel and replaces the user message; status and code stay.

Do **not** wrap a `srvcerror.E` in `fmt.Errorf("%w", err)` before returning it to HTTP or to another service.
Handlers type-assert `srvcerror.E` / `jsonresp.HttpStatusCoder`.
They do not unwrap.
A `%w` wrap becomes a generic 500 and drops the JSON code.

`fmt.Errorf("get task %s: %w", id, err)` is for **repo** (and other non-HTTP Go), so the service still has a useful `err` to log.

## Repo

Return `error`. Pass it up.
Do not import `srvcerror`.
Do not log.
Do not set HTTP status or write Latvian client text.

Wrap driver errors with `%w` so the service can both log the chain and `errors.Is(err, pgx.ErrNoRows)`:

```go
return fmt.Errorf("get task %s: %w", shortID, err)
```

Use `pgx.ErrNoRows` (or a small repo sentinel) for missing rows.
Do not map “not found” to a 404 here.
Repo does not know the client.

## Service

This layer logs the real internal reason and decides what the client (or the calling service) may see.

The facade returns `srvcerror.E` only.

**Unexpected** (Postgres, object store, bugs): log the wrapped `error` with the request logger (`module`, `layer`, operation).
Then return `srvcerror.InternalServerError()` — generic Latvian text, code `internal_server_error`, status 500.
That is the sanitized value.
Do not invent a second internal code per operation (`failed_to_get_task_from_db`).
Do not put the Go wrap string in the JSON message.

**Expected client conditions** (not found, conflict, validation): return an exported sentinel.
Do not log those at error.
The HTTP client already gets `code` + `message`.

```go
var ErrTaskNotFound = srvcerror.New(
	"task_not_found",
	"uzdevums netika atrasts",
).SetHttpStatusCode(http.StatusNotFound)
```

If the message needs data, a small func annotates the sentinel:

```go
func errTaskNotFound(id string) srvcerror.E {
	return ErrTaskNotFound.WithMsg(fmt.Sprintf("uzdevums '%s' netika atrasts", id))
}
```

Translate repo `error` here:

- `errors.Is(err, pgx.ErrNoRows)` → not-found sentinel, no error log
- anything else → log the cause + `InternalServerError()`

Codes name the **condition** (`task_not_found`, `image_dimensions`), not the failed call.
Fixed messages are vars; funcs exist only to `WithMsg`.
500s are `srvcerror.InternalServerError()`.

Task module [`modules/task/srvc/error.go`](../modules/task/srvc/error.go) follows the sentinel/`WithMsg` shape.
Its service logs only unexpected failures, HTTP passes service errors through `WriteError`, and repo wrap strings omit `failed to`.
Other modules still mix `newErr*` helpers and `ErrCode*` consts; do not copy that for new work.

## HTTP

Marshall JSON, auth, cache.
Pass the sanitized service error through.
Do not re-log it.
Do not re-decide service conditions.

Typed handlers return `jsonresp.HttpStatusCoder`.
`srvcerror.E` already implements that.
[`common/httpfunc`](../common/httpfunc/httpfunc.go) calls `jsonresp.WriteError`.

Log in HTTP only for errors that happen **here**: malformed JSON, missing path param, auth failure, failure to write the response.
Use `jsonresp` sentinels for those (`ErrHttpBadRequest.WithMsg`, `BadRequest`, `Forbidden`).
Do not build a `srvcerror` for a problem the service never saw.

`jsonresp.HandleSrvcError` currently `Warn`-logs every service error.
That duplicates (and for 500s, is weaker than) the service log.
Prefer `WriteError` after the service has already logged.

JSON body:

```json
{"status":"error","code":"task_not_found","message":"uzdevums 'sum' netika atrasts"}
```

`code` is the API. Do not rename a published code without treating it as a breaking change.
`message` is Latvian for humans.

## Cross-module

Another service receives the same sanitized `srvcerror.E`, not the Postgres wrap.

Compare with `errors.Is(err, tasksrvc.ErrTaskNotFound)` or `srvcerror.Is(err, "task_not_found")`.
If it is already a client sentinel or `InternalServerError()`, pass it on.
Do not log it again as if this module found the root cause.
Log only a **new** unexpected failure in this service (its own repo, its own store).

Export a sentinel if another package must branch on it.
Keep interpolation in an unexported func so the JSON message can include the id without breaking `Is`.
