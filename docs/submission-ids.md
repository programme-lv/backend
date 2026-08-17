# Submission identifiers

Public URLs use `submissions.short_id` (6-character base62), not `uuid`.

- Generate in `StoreSubm` (`crypto/rand`, retry on `submissions_short_id_key`).
- `GET /subm/{id}` accepts short id or UUID (`parseSubmPathID`).
- JSON `id` is the short id; `subm_uuid` remains the UUID.
- Eval/exec IDs stay UUIDs.

Project-wide note: `../docs/github/submission-ids.md`.
