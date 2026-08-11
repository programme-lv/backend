ALTER TABLE user_email_tokens
	ADD COLUMN sent_at TIMESTAMPTZ;

-- Existing rows were written after a successful send (pre-insert-before-send).
UPDATE user_email_tokens
SET sent_at = created_at
WHERE sent_at IS NULL;
