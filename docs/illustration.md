# Task illustrations and statement images

Uploaded illustration bytes are the original. Existing production files are originals.
The public URL in task JSON is not that file.

## Illustration keys

DB stores `{sha256}{ext}` without a directory prefix.

| Role | Filestore key |
| --- | --- |
| Original | `illustrations/{sha256}{ext}` |
| List (fit 160×160) | `illustrations/{sha256}{ext}.list.webp` |
| View (fit 256×256) | `illustrations/{sha256}{ext}.view.webp` |
| Full (original pixels, WebP) | `illustrations/{sha256}{ext}.full.webp` |

If the original is already `.webp`, `full` is the original object. List and view are still derived when the original is larger than the box.

JSON:

- `http_url` — full WebP
- `list_http_url` — list
- `view_http_url` — task page / admin preview

`width_px`, `height_px`, `sz_in_bytes` are the original.

## Statement image keys

Statement images are stored at `md-images/<taskId>/<sha256-prefix>-<filename>`
(older installs use `task-md-images/` or `task/`). They expose only the full
variant: original pixels as WebP, with no resizing. The task JSON `http_url`
points at `<key>.full.webp`.

- PNG, JPEG, and WebP originals are transcoded.
- SVG and other formats are not transcoded; `http_url` stays the original key.
- The un-suffixed original URL still serves the original bytes.
- Existing originals are derived on first GET; statement image uploads do not
  wait for the variant (unlike illustration uploads, which warm all three).

## Allowlist

`GET /assets/*` is unauthenticated. Variants are a closed suffix set (`list`, `view`, `full`).
Only asset roots `illustrations/` (flat), `md-images/`, `task-md-images/`, and `task/`
are derivable, and only for PNG/JPEG/WebP originals. Unknown keys such as
`.w999.webp` are 404 and never written. Query parameters are not a size API.

## Generation

New uploads warm all three variants.
Existing originals are derived on first GET of that variant (`singleflight`, at most two encodes at a time).
Encoder is pure Go (`github.com/KarpelesLab/gowebp`, quality 80, method 4) because the image is built with `CGO_ENABLED=0`.
Resize does not upscale. Write is temp file + rename.

Original URLs without a variant suffix still serve the original bytes.
