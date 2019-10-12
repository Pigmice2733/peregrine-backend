BEGIN;
ALTER TABLE reports ADD COLUMN comment TEXT;
UPDATE reports SET comment = comments.comment
    FROM comments WHERE
        reports.match_key = comments.match_key
        AND reports.team_key = comments.team_key
        AND reports.reporter_id IS NOT DISTINCT FROM comments.reporter_id
        AND reports.realm_id IS NOT DISTINCT FROM comments.realm_id;
DROP TABLE comments;

ALTER TABLE alliances DROP CONSTRAINT alliances_match_key_fkey;
ALTER TABLE alliances ADD COLUMN event_key TEXT;
UPDATE alliances SET event_key = split_part(match_key, '_', 1);
ALTER TABLE alliances ALTER COLUMN event_key SET NOT NULL;
ALTER TABLE alliances DROP CONSTRAINT alliances_pkey;
ALTER TABLE alliances ADD PRIMARY KEY(event_key, match_key, is_blue);
UPDATE alliances SET match_key = split_part(match_key, '_', 2);

ALTER TABLE reports DROP CONSTRAINT reports_match_key_fkey;
ALTER TABLE reports DROP CONSTRAINT reports_event_key_fkey;
UPDATE reports SET match_key = split_part(match_key, '_', 2);

ALTER TABLE matches DROP CONSTRAINT matches_pkey;
UPDATE matches SET key = split_part(key, '_', 2);
ALTER TABLE matches ADD PRIMARY KEY (key, event_key);

ALTER TABLE reports ADD FOREIGN KEY(event_key, match_key) REFERENCES matches(event_key, key);
ALTER TABLE alliances ADD FOREIGN KEY(event_key, match_key) REFERENCES matches(event_key, key);
COMMIT;
