DROP TABLE IF EXISTS user_email_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS email_verified;
