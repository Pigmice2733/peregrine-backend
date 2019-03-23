CREATE TABLE IF NOT EXISTS comments (
    id SERIAL PRIMARY KEY,
    match_key TEXT NOT NULL REFERENCES matches,
    team_key TEXT NOT NULL,
    reporter_id INTEGER REFERENCES users ON DELETE SET NULL,
    realm_id INTEGER REFERENCES realms ON DELETE SET NULL,
    comment TEXT NOT NULL,

    UNIQUE(match_key, team_key, reporter_id)
);
