BEGIN;
UPDATE reports SET comment = '' WHERE comment IS NULL;
ALTER TABLE reports ALTER COLUMN comment SET NOT NULL;
COMMIT;