CREATE TABLE task_solutions (
    task_id TEXT NOT NULL,
    fname VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    subtasks JSONB NOT NULL DEFAULT '[]',
    PRIMARY KEY (task_id, fname),
    CONSTRAINT fk_solutions_task FOREIGN KEY (task_id) REFERENCES tasks(short_id),
    CONSTRAINT check_fname_length CHECK (LENGTH(fname) <= 255),
    CONSTRAINT check_content_length CHECK (LENGTH(content) <= 1000000),
    CONSTRAINT check_fname_not_empty CHECK (LENGTH(TRIM(fname)) > 0),
    CONSTRAINT check_content_not_empty CHECK (LENGTH(TRIM(content)) > 0)
);
