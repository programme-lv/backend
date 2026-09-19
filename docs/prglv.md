# prglv administration CLI

`prglv` is the admin CLI for the programme.lv backend. It talks to a running
server over the admin HTTP API.

```text
prglv upload <task>      package a TaskZip directory or .zip and import it
prglv export <task-id>   download a task as a TaskZip .zip archive
prglv completions fish   print a shell completion script (fish, bash, zsh)
prglv version
prglv help
```

## Install

```bash
go install github.com/programme-lv/backend/cmd/prglv@latest
```

Or build from the repo root:

```bash
go build -o prglv ./cmd/prglv
```

## Configuration

`upload` and `export` need the backend base URL and the server-to-server admin
key:

| Setting | Flag | Environment |
| --- | --- | --- |
| Backend base URL | `--api-url` | `PRGLV_API_URL`, then `API_PUBLIC_BASE_URL`, default `http://localhost:8080` |
| Admin API key | `--api-key` | `PRGLV_API_KEY`, then `ADMIN_API_KEY` |

Flags win over the environment. The key is sent as
`Authorization: Bearer <key>` and is never logged.

## Upload

```bash
prglv upload path/to/task
prglv upload path/to/task.zip
prglv upload path/to/task --override-id task-copy
prglv upload path/to/task --json
```

`<task>` can be a TaskZip directory or a `.zip` archive. The CLI validates the
package locally with the same reader the backend uses, so layout and metadata
errors surface before anything is sent. It then posts the archive to
`POST /tasks/upload` as the multipart field `task_zip`. `--override-id` sets
the `override_id` query parameter, which stores the task under a different ID
than the archive declares.

When `<task>` is a directory, the CLI builds a flat ZIP from its contents.
The archive may be flat or wrapped in a single directory named after the task
ID; both are accepted. The following top-level entries are never packaged
because the backend ignores them or rejects them outright:

- `archive/`, `testspec/` — accepted but ignored during import;
- `.taskzip/` — locally generated caches;
- `.git/`, `target/`, `__MACOSX/`, and `.DS_Store` — repository or OS noise.

Everything else is included, so a directory that already holds `task.toml`,
`tests/`, `statement/`, and `solutions/` at its root uploads as-is.

## Export

```bash
prglv export lio2026cuska
prglv export lio2026cuska --out /tmp/task.zip
prglv export lio2026cuska --out - > task.zip
prglv export lio2026cuska --force
```

The default output is `<task-id>.zip` in the current directory. Use `-` to
write to stdout. An existing file is not overwritten without `--force`.

## Shell completions

Fish:

```fish
mkdir -p ~/.config/fish/completions
prglv completions fish > ~/.config/fish/completions/prglv.fish
```

Bash and zsh:

```bash
prglv completions bash > /etc/bash_completion.d/prglv
prglv completions zsh > "${fpath[1]}/_prglv"
```

The completion scripts are generated from the same command and flag
definitions as the help text, so they stay in sync.
