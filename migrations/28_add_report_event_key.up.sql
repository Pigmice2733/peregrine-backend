BEGIN;
ALTER TABLE reports ADD COLUMN event_key TEXT;
UPDATE reports SET event_key = split_part(match_key, '_', 1);
ALTER TABLE reports ALTER COLUMN event_key SET NOT NULL;
ALTER TABLE reports ADD CONSTRAINT reports_event_key_fkey FOREIGN KEY (event_key) REFERENCES events(key);
ALTER TABLE reports DROP CONSTRAINT reports_match_key_team_key_reporter_id_key;
ALTER TABLE reports ADD CONSTRAINT reports_event_key_match_key_team_key_reporter_id_key UNIQUE(event_key, match_key,team_key,reporter_id);
COMMIT;