CREATE TABLE task_solution_profiles (
    id UUID PRIMARY KEY,
    task_id TEXT NOT NULL,
    fname VARCHAR(255) NOT NULL,
    lang_id TEXT NOT NULL,
    status TEXT NOT NULL,
    error TEXT,
    expected_points INTEGER NOT NULL,
    possible_points INTEGER NOT NULL,
    tests_version INTEGER NOT NULL,
    solution_sha256 TEXT NOT NULL,
    runs JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ,
    -- Do not reference task_solutions: task updates delete and re-insert all
    -- solution rows, which would cascade away every profile. Profiles for a
    -- removed solution are simply not listed by the admin list endpoint.
    CONSTRAINT fk_profile_task FOREIGN KEY (task_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

CREATE INDEX idx_task_solution_profiles_latest
    ON task_solution_profiles (task_id, fname, created_at DESC);
