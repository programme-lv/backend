# Task illustrations and statement images

Uploaded illustration bytes are the original. Existing production files are originals.
The public URL in task JSON is not that file.

## Illustration keys

DB stores the SHA-256 and extension separately. Paths are derived from task ID
and metadata; see [task storage](task-storage.md).

| Role | Filestore key | Public URL |
| --- | --- | --- |
| Original | `assets/tasks/{taskId}/illustration/{sha256}{ext}` | `.../assets/tasks/...` |
| List (fit 160×160) | `cache/images/{sha256(originalKey)}/list.webp` | `<original>.list.webp` |
| View (fit 256×256) | `cache/images/{sha256(originalKey)}/view.webp` | `<original>.view.webp` |
| Full (original pixels, WebP) | `cache/images/{sha256(originalKey)}/full.webp` | `<original>.full.webp` |

Public URLs keep the stable suffix form; the server derives the cache object on
demand. If the original is already `.webp`, `full` is the original object.
List and view are still derived when the original is larger than the box.

JSON:

- `http_url` — full WebP
- `list_http_url` — list
- `view_http_url` — task page / admin preview

`width_px`, `height_px`, `sz_in_bytes` are the original.

## Statement image keys

Statement images are stored at
`assets/tasks/<taskId>/images/<sha256>/<filename>`. The DB stores filename,
full SHA-256 and dimensions. They expose only the full variant: original pixels
as WebP in `cache/`, with no resizing. The task JSON `http_url` uses the
`<original>.full.webp` URL (or the original for a `.webp` source); the handler
generates the cache object on first request.

- PNG, JPEG, and WebP originals are transcoded.
- SVG and other formats are not transcoded; `http_url` stays the original key.
- The un-suffixed original URL still serves the original bytes.
- Existing originals are derived on first GET; statement image uploads do not
  wait for the variant (unlike illustration uploads, which warm all three).

## Allowlist

`GET /assets/*` is unauthenticated. It serves `assets/tasks/...` originals and
`cache/images/...` derivatives. Legacy `.list/.view/.full.webp` URLs under
`assets/tasks/...` are generated into cache on demand. Task `archive/` files
are admin-only and are not served by this route. Only PNG/JPEG/WebP originals
are derivable. Unknown keys such as `.w999.webp` are 404 and never written.
Query parameters are not a size API.

## Generation

New uploads warm all three variants.
Existing originals are derived on first GET of that variant (`singleflight`, at most two encodes at a time).
Encoder is pure Go (`github.com/KarpelesLab/gowebp`, quality 80, method 4) because the image is built with `CGO_ENABLED=0`.
Resize does not upscale. Write is temp file + rename.

Original URLs without a variant suffix still serve the original bytes.
