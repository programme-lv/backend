# Submission list

`GET /subm` returns a paginated list (`page` + `pagination`). Default `limit` 30, max 100.

Query params:

| Param | Effect |
| --- | --- |
| `offset`, `limit` | Pagination |
| `search` | Fuzzy **OR** across task name/id, username/uuid, language name/id (max 100 chars) |
| `task_id` | Exact task short id **AND** (`submissions.task_shortid`, max 50 chars) |
| `mine=1` | AND current JWT user. **401** if unauthenticated |

`task_id` and `mine` are independent AND filters. They are not part of `search`. Do not pass a task id as `search` to mean “this task only”.

The list cache key includes author UUID and `task_id` so `mine` pages are not shared across users.

Public `id` vs `subm_uuid`: [submission-ids.md](submission-ids.md).
