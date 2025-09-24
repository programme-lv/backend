-- Migration: 001_initial_migration.up.sql

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table: submissions
CREATE TABLE submissions (
    subm_uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    content TEXT NOT NULL,
    author_uuid UUID NOT NULL,
    task_id TEXT NOT NULL,
    prog_lang_id TEXT NOT NULL,
    current_eval_uuid UUID,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table: evaluations
CREATE TABLE evaluations (
    eval_uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    subm_uuid UUID REFERENCES submissions(subm_uuid) ON DELETE CASCADE,
    evaluation_stage VARCHAR(20) NOT NULL CHECK (evaluation_stage IN ('waiting', 'received', 'compiling', 'testing', 'finished', 'error')),
    scoring_method VARCHAR(20) NOT NULL CHECK (scoring_method IN ('tests', 'subtask', 'testgroup')),
    cpu_time_limit_millis INTEGER,
    mem_limit_kibi_bytes INTEGER,
    error_message TEXT,
    testlib_checker_code TEXT NOT NULL,
    system_information TEXT,
    subm_compile_stdout TEXT,
    subm_compile_stderr TEXT,
    subm_compile_exit_code INTEGER,
    subm_compile_cpu_time_millis INTEGER,
    subm_compile_wall_time_millis INTEGER,
    subm_compile_memory_kibi_bytes INTEGER,
    subm_compile_ctx_switches_forced BIGINT,
    subm_compile_exit_signal BIGINT,
    subm_compile_isolate_status VARCHAR(50),
    programming_lang_id TEXT NOT NULL,
    programming_lang_display_name TEXT NOT NULL,
    programming_lang_subm_code_fname TEXT NOT NULL,
    programming_lang_compile_command TEXT,
    programming_lang_compiled_fname TEXT,
    programming_lang_exec_command TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Table: evaluation_tests
CREATE TABLE evaluation_tests (
    eval_uuid UUID REFERENCES evaluations(eval_uuid) ON DELETE CASCADE,
    test_id INTEGER NOT NULL,
    full_input_s3_url TEXT NOT NULL,
    full_answer_s3_url TEXT NOT NULL,
    reached BOOLEAN NOT NULL DEFAULT FALSE,
    ignored BOOLEAN NOT NULL DEFAULT FALSE,
    finished BOOLEAN NOT NULL DEFAULT FALSE,
    input_trimmed TEXT,
    answer_trimmed TEXT,
    checker_stdout TEXT,
    checker_stderr TEXT,
    checker_exit_code INTEGER,
    checker_cpu_time_millis INTEGER,
    checker_wall_time_millis INTEGER,
    checker_memory_kibi_bytes INTEGER,
    checker_ctx_switches_forced BIGINT,
    checker_exit_signal BIGINT,
    checker_isolate_status VARCHAR(50),
    subm_stdout TEXT,
    subm_stderr TEXT,
    subm_exit_code INTEGER,
    subm_cpu_time_millis INTEGER,
    subm_wall_time_millis INTEGER,
    subm_memory_kibi_bytes INTEGER,
    subm_ctx_switches_forced BIGINT,
    subm_exit_signal INTEGER,
    subm_isolate_status VARCHAR(50),
    subtasks INTEGER[] DEFAULT ARRAY[]::INTEGER[],
    testgroups INTEGER[] DEFAULT ARRAY[]::INTEGER[],
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (eval_uuid, test_id)
);

-- Table: evaluation_scoring_subtasks
CREATE TABLE evaluation_scoring_subtasks (
    eval_uuid UUID REFERENCES evaluations(eval_uuid) ON DELETE CASCADE,
    subtask_id INTEGER NOT NULL,
    subtask_points INTEGER NOT NULL DEFAULT 0,
    accepted INTEGER NOT NULL DEFAULT 0,
    wrong INTEGER NOT NULL DEFAULT 0,
    untested INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (eval_uuid, subtask_id)
);

-- Table: evaluation_scoring_testgroups
CREATE TABLE evaluation_scoring_testgroups (
    eval_uuid UUID REFERENCES evaluations(eval_uuid) ON DELETE CASCADE,
    testgroup_id INTEGER NOT NULL,
    statement_subtasks INTEGER[] DEFAULT ARRAY[]::INTEGER[],
    testgroup_points INTEGER NOT NULL DEFAULT 0,
    accepted INTEGER NOT NULL DEFAULT 0,
    wrong INTEGER NOT NULL DEFAULT 0,
    untested INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (eval_uuid, testgroup_id)
);
