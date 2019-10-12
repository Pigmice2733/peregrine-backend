BEGIN;
ALTER TABLE reports DROP CONSTRAINT reports_event_key_match_key_team_key_reporter_id_key;
ALTER TABLE reports DROP COLUMN event_key;
ALTER TABLE reports ADD CONSTRAINT reports_match_key_team_key_reporter_id_key UNIQUE(match_key,team_key,reporter_id);
COMMIT;