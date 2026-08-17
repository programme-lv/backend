ALTER TABLE submissions
	ADD COLUMN short_id VARCHAR(6);

DO $$
DECLARE
	r RECORD;
	chars TEXT := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
	candidate TEXT;
	i INT;
BEGIN
	FOR r IN SELECT uuid FROM submissions WHERE short_id IS NULL LOOP
		LOOP
			candidate := '';
			FOR i IN 1..6 LOOP
				candidate := candidate || substr(chars, 1 + (floor(random() * 62))::int, 1);
			END LOOP;
			EXIT WHEN candidate <> 'scores'
				AND NOT EXISTS (SELECT 1 FROM submissions WHERE short_id = candidate);
		END LOOP;
		UPDATE submissions SET short_id = candidate WHERE uuid = r.uuid;
	END LOOP;
END $$;

ALTER TABLE submissions
	ALTER COLUMN short_id SET NOT NULL;

ALTER TABLE submissions
	ADD CONSTRAINT submissions_short_id_key UNIQUE (short_id);

ALTER TABLE submissions
	ADD CONSTRAINT submissions_short_id_format
	CHECK (short_id ~ '^[0-9A-Za-z]{6}$' AND short_id <> 'scores');
