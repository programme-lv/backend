ALTER TABLE users
    ADD COLUMN IF NOT EXISTS pwd_changed_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE users SET pwd_changed_at = created_at;
