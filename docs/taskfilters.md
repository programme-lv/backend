# Task list origin filters

`GET /task-filters` returns the catalog of origin combinations that have tasks: olympiad → year → stage → division, with counts.

It is a separate path from `GET /tasks` on purpose:

- The list is task previews (names, images, origin), capped at 100.
- The tree is every distinct origin tuple in `tasks`, including rows beyond that cap.
- Chi would treat `GET /tasks/filters` as `{taskId}=filters`.

Empty/`NULL` `origin_olympiad` becomes the UI bucket `"other"`. That id is not stored in the column.

`origin_year` is normalized like LIO edition year: `"2024/2025"` → `"2025"`. `GET /tasks` previews use the same olympiad and year ids (`origin_olympiad`, `origin_year`). `olymp_stage` is stored as-is. `origin_divisions` stays the stored array; `"both"` exists only in this tree (and when the website matches `["junior","senior"]`).

If the archive grows past 100 tasks, a selected leaf can show zero cards until list pagination exists.

The additive `sources` catalog uses `{id, kind, count, years}`. A task's
`source_id` is its nonblank stored olympiad first, otherwise `PPS` if its
organization is PPS (case-insensitive, trimmed), otherwise `other`. The same
rule is used in previews, task detail and source counts, so an LIO task with a
PPS organization remains LIO. Kinds are `olympiad`, `institution` and `other`;
PPS and other have empty years. Source counts partition all tasks, while the
legacy `olympiads` catalog still groups all blank olympiads under `other`,
including their year hierarchy. Existing origin fields retain their prior
meaning and normalization. LIO is first, other last, and remaining IDs sort
lexically.

Task edits invalidate list, detail and filter caches. Other changes outside the
edit endpoint can take up to the existing 20-second cache TTL to appear.
