ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email_verified BOOLEAN NOT NULL DEFAULT false;

UPDATE users SET email_verified = true WHERE email_verified = false;

CREATE TABLE IF NOT EXISTS user_email_tokens (
    uuid UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_uuid UUID NOT NULL REFERENCES users(uuid) ON DELETE CASCADE,
    purpose TEXT NOT NULL CHECK (purpose IN ('password_reset', 'email_verify')),
    token_hash TEXT NOT NULL UNIQUE,
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS user_email_tokens_user_purpose_created_idx
    ON user_email_tokens (user_uuid, purpose, created_at DESC);
