-- Create the main tasks table.
CREATE TABLE IF NOT EXISTS tasks (
    short_id TEXT PRIMARY KEY,
    full_name TEXT NOT NULL,
    illustr_img_url TEXT,
    mem_lim_megabytes INTEGER NOT NULL,
    cpu_time_lim_secs DOUBLE PRECISION NOT NULL,
    origin_olympiad TEXT,
    difficulty_rating INTEGER,
    checker TEXT,
    interactor TEXT
);

-- Create table for origin notes.
CREATE TABLE IF NOT EXISTS task_origin_notes (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    lang TEXT NOT NULL,
    info TEXT,
    CONSTRAINT fk_origin_notes_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- Create table for Markdown statements.
CREATE TABLE IF NOT EXISTS task_md_statements (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    lang_iso639 TEXT,
    story TEXT,
    input TEXT,
    output TEXT,
    notes TEXT,
    scoring TEXT,
    talk TEXT,
    example TEXT,
    CONSTRAINT fk_md_statements_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- Create table for images used in Markdown statements.
CREATE TABLE IF NOT EXISTS task_md_statement_images (
    id SERIAL PRIMARY KEY,
    md_statement_id INTEGER NOT NULL,
    uuid TEXT,
    s3_url TEXT,
    width_px INTEGER,
    height_px INTEGER,
    width_em INTEGER,
    CONSTRAINT fk_md_statement_images_md_statement FOREIGN KEY (md_statement_id)
        REFERENCES task_md_statements(id) ON DELETE CASCADE
);

-- Create table for PDF statements.
CREATE TABLE IF NOT EXISTS task_pdf_statements (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    lang_iso639 TEXT,
    object_url TEXT,
    CONSTRAINT fk_pdf_statements_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- Create table for visible input subtasks.
CREATE TABLE IF NOT EXISTS task_vis_inp_subtasks (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    external_subtask_id INTEGER,  -- from JSON field "SubtaskId"
    CONSTRAINT fk_vis_inp_subtasks_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE,
    CONSTRAINT unique_task_subtask UNIQUE (task_short_id, external_subtask_id)
);

-- Create table for tests belonging to a visible input subtask.
CREATE TABLE IF NOT EXISTS task_vis_inp_subtask_tests (
    id SERIAL PRIMARY KEY,
    subtask_id INTEGER NOT NULL,  -- references task_vis_inp_subtasks(id)
    test_id INTEGER,              -- from JSON field "TestId"
    input TEXT,
    CONSTRAINT fk_vis_inp_subtask_tests FOREIGN KEY (subtask_id)
        REFERENCES task_vis_inp_subtasks(id) ON DELETE CASCADE
);

-- Create table for examples.
CREATE TABLE IF NOT EXISTS task_examples (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    input TEXT,
    output TEXT,
    md_note TEXT,
    CONSTRAINT fk_examples_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- Create table for tests (evaluation).
CREATE TABLE IF NOT EXISTS task_tests (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    inp_sha2 TEXT,
    ans_sha2 TEXT,
    CONSTRAINT fk_tests_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- (Optional) Create table for scoring subtasks.
CREATE TABLE IF NOT EXISTS task_subtasks (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    score INTEGER,
    descriptions JSONB,  -- storing map of language => description
    CONSTRAINT fk_subtasks_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- (Optional) Create join table for mapping subtasks to test IDs.
CREATE TABLE IF NOT EXISTS task_subtask_test_ids (
    id SERIAL PRIMARY KEY,
    subtask_id INTEGER NOT NULL,
    test_id INTEGER NOT NULL,
    CONSTRAINT fk_subtask_test_ids_subtask FOREIGN KEY (subtask_id)
        REFERENCES task_subtasks(id) ON DELETE CASCADE
);

-- (Optional) Create table for test groups.
CREATE TABLE IF NOT EXISTS task_test_groups (
    id SERIAL PRIMARY KEY,
    task_short_id TEXT NOT NULL,
    points INTEGER,
    public BOOLEAN,
    CONSTRAINT fk_test_groups_task FOREIGN KEY (task_short_id)
        REFERENCES tasks(short_id) ON DELETE CASCADE
);

-- (Optional) Create join table for mapping test groups to test IDs.
CREATE TABLE IF NOT EXISTS task_test_group_test_ids (
    id SERIAL PRIMARY KEY,
    test_group_id INTEGER NOT NULL,
    test_id INTEGER NOT NULL,
    CONSTRAINT fk_test_group_test_ids FOREIGN KEY (test_group_id)
        REFERENCES task_test_groups(id) ON DELETE CASCADE
);
