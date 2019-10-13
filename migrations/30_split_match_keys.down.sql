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

CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    event_key TEXT NOT NULL REFERENCES events,
    match_key TEXT NOT NULL REFERENCES matches,
    team_key TEXT NOT NULL,
    reporter_id INTEGER REFERENCES users ON DELETE SET NULL,
    realm_id INTEGER REFERENCES realms ON DELETE SET NULL,
    comment TEXT NOT NULL,

    UNIQUE(event_key, match_key, team_key, reporter_id)
);

INSERT INTO comments (event_key, match_key, team_key, reporter_id, realm_id, comment)
    SELECT event_key, match_key, team_key, reporter_id, realm_id, comment
    FROM reports WHERE comment IS NOT NULL;

ALTER TABLE reports DROP COLUMN comment;

COMMIT;
