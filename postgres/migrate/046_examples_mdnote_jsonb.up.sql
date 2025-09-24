-- Make example markdown notes translatable via JSONB
ALTER TABLE task_examples
ALTER COLUMN md_note TYPE jsonb
USING (
  CASE
    WHEN md_note IS NULL OR md_note = '' THEN '{}'::jsonb
    ELSE jsonb_build_object('lv', md_note)
  END
);

