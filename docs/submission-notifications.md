# Submission activity notifications

The backend can send a phone notification through [ntfy](https://ntfy.sh) when
activity resumes after a quiet period. This is useful because programme.lv sees
long gaps between submissions.

## Behaviour

- A notification is considered only after a submission has been stored
  successfully. Failures to notify never fail the submission.
- The previous non-admin submission time is read before the new submission is
  stored. If the gap is at least `NTFY_QUIET_PERIOD` (default `6h`), one
  notification is published. If there is no earlier submission, the first one
  counts as resumed activity.
- Submissions by the reserved `admin` account are ignored, both as the current
  author and as the previous activity. This keeps admin testing from waking the
  phone.
- A process-local guard prevents a burst of concurrent submissions from sending
  more than one notification per quiet period. Like the email rate limiter, this
  assumes a single backend instance.
- The message contains the username, task short ID, language short ID, and the
  quiet duration. It never contains source code. When `WEBSITE_PUBLIC_BASE_URL`
  is set, the notification links to `/submissions/{short_id}`.
- Publishing runs in the background with a 10-second timeout, so the submit
  request does not wait for ntfy.

The implementation lives in [`modules/subm/notify`](../modules/subm/notify) and
[`modules/subm/srvc/activity.go`](../modules/subm/srvc/activity.go).

## Env

| Variable | Default | Notes |
| --- | --- | --- |
| `NTFY_ENABLED` | `false` | Enables submission activity notifications |
| `NTFY_TOPIC` | empty | Required when enabled; 1–64 chars of `A-Z a-z 0-9 - _` |
| `NTFY_SERVER_URL` | `https://ntfy.sh` | Any ntfy-compatible server |
| `NTFY_TOKEN` | empty | Optional bearer token for protected topics |
| `NTFY_QUIET_PERIOD` | `6h` | Positive Go duration |
| `WEBSITE_PUBLIC_BASE_URL` | empty | Used for the notification click target |

Treat the topic as a secret. Topics on the public ntfy.sh server can be read and
published to by anyone who knows the name, so use a long random suffix and keep
the production value in the deployment `.env`, not in this repository.

## Verifying locally

1. Subscribe to the topic in the ntfy app or with
   `curl -s https://ntfy.sh/<topic>/json`.
2. Set `NTFY_ENABLED=true`, `NTFY_TOPIC=<topic>`, and a short
   `NTFY_QUIET_PERIOD=1s` in the backend `.env`.
3. Submit a solution from a non-admin user. The first submission after the
   period should produce one notification; immediate follow-ups should not.

The ntfy HTTP API is documented at <https://docs.ntfy.sh/publish/>.
