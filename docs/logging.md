# Logging

Production monitoring is warnings and errors. Info is for process lifecycle and unusual successful operations, not per-request or per-test chatter.

## Level

| Situation | Level |
| --- | --- |
| No `.env` (container / production) | `warn` |
| `.env` loaded (local development) | `info` |
| `LOG_LEVEL` set | that value |

`LOG_LEVEL` accepts `debug` (`dbg`), `info`, `warn` (`warning`), `error`. Invalid values abort startup.

## What belongs where

- **Error**: unexpected failures already covered by [errors.md](errors.md).
- **Warn**: client-origin HTTP failures, status ≥ 400, slow requests (≥500ms, except SSE `/subm-updates`).
- **Info**: server start/stop, migrations actually applied, one-off operational jobs.
- **Debug**: handler enter/exit, cache hits, eval event stream, per-test progress, object-store saves, disabled-SMTP skips.

Do not log “getting X” / “returning X” around HTTP handlers. The request logger already records failures and slowness; Prometheus covers traffic shape.
