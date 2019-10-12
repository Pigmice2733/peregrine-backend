BEGIN;
ALTER TABLE alliances DROP CONSTRAINT alliances_event_key_match_key_fkey;
UPDATE alliances SET match_key = format('%s_%s', event_key, match_key);
ALTER TABLE alliances DROP CONSTRAINT alliances_pkey;
ALTER TABLE alliances DROP COLUMN event_key;

ALTER TABLE reports DROP CONSTRAINT reports_event_key_match_key_fkey;
UPDATE reports SET match_key = format('%s_%s', event_key, match_key);

ALTER TABLE matches DROP CONSTRAINT matches_pkey;
UPDATE matches SET key = format('%s_%s', event_key, key);
ALTER TABLE matches ADD PRIMARY KEY (key);

ALTER TABLE reports ADD FOREIGN KEY(match_key) REFERENCES matches(key);
ALTER TABLE reports ADD FOREIGN KEY(event_key) REFERENCES events(key);
ALTER TABLE alliances ADD FOREIGN KEY(match_key) REFERENCES matches(key);
ALTER TABLE alliances ADD PRIMARY KEY(match_key, is_blue);

ALTER TABLE comments ADD COLUMN event_key TEXT REFERENCES events;
ALTER TABLE comments ADD COLUMN match_key TEXT REFERENCES matches(key);
ALTER TABLE comments ADD COLUMN team_key TEXT;
ALTER TABLE comments ADD COLUMN reporter_id INTEGER REFERENCES users ON DELETE SET NULL;
ALTER TABLE comments ADD COLUMN realm_id INTEGER REFERENCES realms ON DELETE SET NULL;

UPDATE comments SET
    event_key = reports.event_key,
    match_key = reports.match_key,
    team_key = reports.team_key,
    reporter_id = reports.reporter_id,
    realm_id = reports.realm_id
    FROM reports WHERE
        reports.id = comments.report_id;

ALTER TABLE comments DROP COLUMN report_id;

ALTER TABLE comments ALTER COLUMN event_key SET NOT NULL;
ALTER TABLE comments ALTER COLUMN match_key SET NOT NULL;
ALTER TABLE comments ALTER COLUMN team_key SET NOT NULL;
COMMIT;
