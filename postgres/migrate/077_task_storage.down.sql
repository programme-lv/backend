-- Paths are reconstructed from verified file bytes during startup. Reverting
-- only SQL cannot restore the old storage layout; restore a paired DB/storage
-- backup instead.
DO $$ BEGIN RAISE EXCEPTION 'task storage migration requires restoring a paired database and filestore backup'; END $$;
