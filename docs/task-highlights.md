# Editorial task highlights

Highlights are separate editorial metadata, not TaskZIP or original task fields.
They survive task updates and do not change the task revision or tests version.
Deleting a task cascades its assignments and highlight revision; catalogue entries
are never hard-deleted through the API. Migration 081 adds aliases only to
existing dynamic-programming and breadth-first-search rows, and corrects the old
binary-search label only when it matches the known old spelling. It creates no rows.

All management routes use the existing admin JWT cookie or Bearer admin API key;
anonymous and non-admin callers are rejected. Responses use the normal JSON
`status`/`data` envelope. All labels are trimmed, nonblank and at most 60 Unicode
characters. IDs are lowercase ASCII words separated by single hyphens, at most
48 bytes. Optional `abbreviation` and `english_label` are trimmed, allow empty,
forbid control characters, and are limited to 16 and 80 Unicode characters,
respectively. Empty values are omitted from responses. PATCH omission preserves
each alias independently; an empty string clears it.

- `GET /task-highlight-tags`: all `{id,label,archived,revision,abbreviation?,english_label?}` rows, including retired.
- `POST /task-highlight-tags`: `{id,label,abbreviation?,english_label?}` creates an active row at revision 1;
  duplicate IDs return 409.
- `PATCH /task-highlight-tags/{tagId}`: `{label,archived,expected_revision,abbreviation?,english_label?}`
  atomically changes a row and increments its revision; stale writes return 409.
- `GET /tasks/{taskId}/highlighted-tags`: `{highlighted_tags:[{id,label,abbreviation?,english_label?}],revision}`;
  a task without highlights starts at `[]` and revision 0.
- `PATCH /tasks/{taskId}/highlighted-tags`: `{highlighted_tags:[id,...],expected_revision}`
  replaces the ordered list with zero to two unique IDs. An empty array clears it;
  missing or null arrays are invalid. Stale writes return 409. Unknown IDs and
  newly assigned retired IDs are rejected; already assigned retired IDs may be
  retained or reordered. Every successful replacement increments the revision.

Retired assignments remain visible until removed. Public task list and task view
responses include `highlighted_tags` when nonempty, with current catalogue labels and nonempty aliases.
Labels are fetched in one batch after reading cached task data; database failures
return an internal error rather than stale labels.
