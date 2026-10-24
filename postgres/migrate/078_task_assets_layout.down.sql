-- Asset paths are derived from the current layout. Reverting requires moving
-- directories back and clearing the marker; there is no SQL-only rollback.
DO $$ BEGIN RAISE EXCEPTION 'task asset layout migration requires restoring a paired database and filestore backup'; END $$;
