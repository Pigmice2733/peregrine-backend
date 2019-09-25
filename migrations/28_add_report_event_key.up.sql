BEGIN;
ALTER TABLE reports ADD COLUMN event_key TEXT;
UPDATE reports SET event_key = split_part(match_key, '_', 1);
ALTER TABLE reports ADD CONSTRAINT reports_event_key_fkey FOREIGN KEY (event_key) REFERENCES events(key);
COMMIT;