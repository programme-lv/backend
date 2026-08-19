# Task list origin filters

`GET /task-filters` returns the catalog of origin combinations that have tasks: olympiad → year → stage → division, with counts.

It is a separate path from `GET /tasks` on purpose:

- The list is task previews (names, images, origin), capped at 100.
- The tree is every distinct origin tuple in `tasks`, including rows beyond that cap.
- Chi would treat `GET /tasks/filters` as `{taskId}=filters`.

Empty/`NULL` `origin_olympiad` becomes the UI bucket `"other"`. That id is not stored in the column.

`origin_year` is normalized like LIO edition year: `"2024/2025"` → `"2025"`. `GET /tasks` previews use the same olympiad and year ids (`origin_olympiad`, `origin_year`). `olymp_stage` is stored as-is. `origin_divisions` stays the stored array; `"both"` exists only in this tree (and when the website matches `["junior","senior"]`).

If the archive grows past 100 tasks, a selected leaf can show zero cards until list pagination exists.
