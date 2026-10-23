-- Optimistic concurrency for all task mutations. Every writer bumps revision.
ALTER TABLE tasks
ADD COLUMN IF NOT EXISTS revision INTEGER NOT NULL DEFAULT 1;
