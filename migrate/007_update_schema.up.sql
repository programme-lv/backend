BEGIN;
-- rename programming_lang_id to language_id
-- +----------------------------------+--------------------------+--------------------------------------+
-- | Column                           | Type                     | Modifiers                            |
-- |----------------------------------+--------------------------+--------------------------------------|
-- | eval_uuid                        | uuid                     |  not null default uuid_generate_v4() |
-- | evaluation_stage                 | character varying(20)    |  not null                            |
-- | scoring_method                   | character varying(20)    |  not null                            |
-- | cpu_time_limit_millis            | integer                  |  not null                            |
-- | mem_limit_kibi_bytes             | integer                  |  not null                            |
-- | error_message                    | text                     |                                      |
-- | system_information               | text                     |                                      |
-- | subm_compile_stdout              | text                     |                                      |
-- | subm_compile_stderr              | text                     |                                      |
-- | subm_compile_exit_code           | integer                  |                                      |
-- | subm_compile_cpu_time_millis     | integer                  |                                      |
-- | subm_compile_wall_time_millis    | integer                  |                                      |
-- | subm_compile_memory_kibi_bytes   | integer                  |                                      |
-- | subm_compile_ctx_switches_forced | bigint                   |                                      |
-- | subm_compile_exit_signal         | bigint                   |                                      |
-- | subm_compile_isolate_status      | character varying(50)    |                                      |
-- | programming_lang_id              | text                     |  not null                            |
-- | programming_lang_display_name    | text                     |  not null                            |
-- | programming_lang_subm_code_fname | text                     |  not null                            |
-- | programming_lang_compile_command | text                     |                                      |
-- | programming_lang_compiled_fname  | text                     |                                      |
-- | programming_lang_exec_command    | text                     |  not null                            |
-- | created_at                       | timestamp with time zone |  not null default now()              |
-- | checker_id                       | integer                  |                                      |
-- +----------------------------------+--------------------------+--------------------------------------+

ALTER TABLE evaluations RENAME COLUMN programming_lang_id TO language_id;
ALTER TABLE evaluations RENAME COLUMN programming_lang_display_name TO language_display_name;
ALTER TABLE evaluations RENAME COLUMN programming_lang_subm_code_fname TO language_subm_code_fname;
ALTER TABLE evaluations RENAME COLUMN programming_lang_compile_command TO language_compile_command;
ALTER TABLE evaluations RENAME COLUMN programming_lang_compiled_fname TO language_compiled_fname;

ALTER TABLE evaluations RENAME COLUMN checker_id TO testlib_checker_id;

-- move the subm_compile data to runtime_data table

-- postgres@database-2:postgres> \d runtime_data
-- +---------------------+--------------------------+------------------------------------------------------------+
-- | Column              | Type                     | Modifiers                                                  |
-- |---------------------+--------------------------+------------------------------------------------------------|
-- | id                  | integer                  |  not null default nextval('runtime_data_id_seq'::regclass) |
-- | eval_uuid           | uuid                     |  not null                                                  |
-- | test_id             | integer                  |  not null                                                  |
-- | stdout              | text                     |                                                            |
-- | stderr              | text                     |                                                            |
-- | exit_code           | integer                  |                                                            |
-- | cpu_time_millis     | integer                  |                                                            |
-- | wall_time_millis    | integer                  |                                                            |
-- | memory_kibi_bytes   | integer                  |                                                            |
-- | ctx_switches_forced | bigint                   |                                                            |
-- | exit_signal         | bigint                   |                                                            |
-- | isolate_status      | character varying(50)    |                                                            |
-- | created_at          | timestamp with time zone |  default now()                                             |
-- +---------------------+--------------------------+------------------------------------------------------------+
-- Indexes:
--     "runtime_data_pkey" PRIMARY KEY, btree (id)
-- Referenced by:
--     TABLE "evaluation_tests" CONSTRAINT "fk_checker_runtime" FOREIGN KEY (checker_runtime_id) REFERENCES runtime_data(id) ON DELETE SET NULL
--     TABLE "evaluation_tests" CONSTRAINT "fk_subm_runtime" FOREIGN KEY (subm_runtime_id) REFERENCES runtime_data(id) ON DELETE SET NULL

-- Time: 0.271s

ALTER TABLE evaluations DROP COLUMN subm_compile_stdout;
ALTER TABLE evaluations DROP COLUMN subm_compile_stderr;
ALTER TABLE evaluations DROP COLUMN subm_compile_exit_code;
ALTER TABLE evaluations DROP COLUMN subm_compile_cpu_time_millis;
ALTER TABLE evaluations DROP COLUMN subm_compile_wall_time_millis;
ALTER TABLE evaluations DROP COLUMN subm_compile_memory_kibi_bytes;
ALTER TABLE evaluations DROP COLUMN subm_compile_ctx_switches_forced;
ALTER TABLE evaluations DROP COLUMN subm_compile_exit_signal;
ALTER TABLE evaluations DROP COLUMN subm_compile_isolate_status;

-- add compile_runtime_id to evaluations
ALTER TABLE evaluations ADD COLUMN compile_runtime_id INTEGER;
COMMIT;
