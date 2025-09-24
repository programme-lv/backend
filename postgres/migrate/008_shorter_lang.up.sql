-- | language_id                   | text                     |  not null                            |
-- | language_display_name         | text                     |  not null                            |
-- | language_subm_code_fname      | text                     |  not null                            |
-- | language_compile_command      | text                     |                                      |
-- | language_compiled_fname       | text                     |                                      |
BEGIN;
ALTER TABLE evaluations RENAME COLUMN language_id TO lang_id;
ALTER TABLE evaluations RENAME COLUMN language_display_name TO lang_name;
ALTER TABLE evaluations RENAME COLUMN language_subm_code_fname TO lang_code_fname;
ALTER TABLE evaluations RENAME COLUMN language_compile_command TO lang_comp_cmd;
ALTER TABLE evaluations RENAME COLUMN language_compiled_fname TO lang_comp_fname;
ALTER TABLE evaluations RENAME COLUMN programming_lang_exec_command TO lang_exec_cmd;
COMMIT;
