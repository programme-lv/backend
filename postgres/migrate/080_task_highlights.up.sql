CREATE TABLE task_highlight_tags (
    id text PRIMARY KEY CHECK (length(id) <= 48 AND id ~ '^[a-z]+(-[a-z]+)*$'),
    label text NOT NULL CHECK (length(btrim(label)) BETWEEN 1 AND 60),
    archived boolean NOT NULL DEFAULT false,
    revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0)
);
CREATE TABLE task_highlight_revisions (
    task_short_id text PRIMARY KEY REFERENCES tasks(short_id) ON DELETE CASCADE,
    revision bigint NOT NULL CHECK (revision > 0)
);
CREATE TABLE task_highlight_assignments (
    task_short_id text NOT NULL REFERENCES tasks(short_id) ON DELETE CASCADE,
    tag_id text NOT NULL REFERENCES task_highlight_tags(id),
    position integer NOT NULL CHECK (position BETWEEN 0 AND 1),
    PRIMARY KEY (task_short_id, tag_id),
    UNIQUE (task_short_id, position)
);
