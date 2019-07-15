CREATE TABLE IF NOT EXISTS all_teams (
    key TEXT PRIMARY KEY,
    nickname TEXT
);

INSERT INTO all_teams (key) SELECT key FROM teams;

ALTER TABLE teams
    ADD CONSTRAINT teams_key_fkey
        FOREIGN KEY (key)
        REFERENCES all_teams (key);
